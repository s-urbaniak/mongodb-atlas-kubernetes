package dryrun

import "errors"

// ErrDryRun is a sentinel error, meant to be introspected using errors.Is
var ErrDryRun = errors.New("dry-run")
