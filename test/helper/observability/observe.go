package observability

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/mongodb/mongodb-atlas-kubernetes/v2/test/helper/jsonwriter"
	"github.com/mongodb/mongodb-atlas-kubernetes/v2/test/helper/loki_reporter"
)

func Observe(cmdArgs ...string) error {
	var (
		lokiURL     string
		lokiEnabled bool
	)

	flag.BoolVar(&lokiEnabled, "loki-enabled", true, "")
	flag.StringVar(&lokiURL, "loki-url", "http://localhost:30002", "")
	flag.Parse()

	target := io.Discard
	if lokiEnabled {
		loki, err := loki_reporter.New(lokiURL, os.Stderr)
		if err != nil {
			return fmt.Errorf("error setting up loki: %w", err)
		}
		target = loki
		defer loki.Stop()

	}

	return forwardTo(target, cmdArgs...)
}

func forwardTo(target io.Writer, cmdArgs ...string) error {
	cmd := exec.Command(cmdArgs[0], append(cmdArgs[1:])...)

	cmdStdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get cmdStdout pipe: %w", err)
	}

	cmdStderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get cmdStdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	for _, dest := range []struct {
		writer io.Writer
		reader io.Reader
	}{
		{
			writer: io.MultiWriter(os.Stdout, jsonwriter.NewWithLevel(target, "info")),
			reader: cmdStdout,
		},
		{
			writer: io.MultiWriter(os.Stderr, jsonwriter.NewWithLevel(target, "error")),
			reader: cmdStderr,
		},
	} {
		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(dest.reader)
			for scanner.Scan() {
				_, err := dest.writer.Write(append(scanner.Bytes(), '\n'))
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					return
				}
			}
		}()
	}
	wg.Wait()

	return nil
}
