package main

import (
	"fmt"
	"os"

	"github.com/mongodb/mongodb-atlas-kubernetes/v2/test/helper/observability"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, `available commands: "install", "snapshot", "local", "observe"`)
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "install":
		err = observability.Install(os.Stdout)
	case "snapshot":
		err = observability.Snapshot(os.Stdout)
	case "local":
		err = observability.InstallLocalDevelopment(os.Stdout, os.Args[2])
	case "observe":
		err = observability.Observe(os.Args[2:]...)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
