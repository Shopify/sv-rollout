package main

import (
	"errors"
	"fmt"
)

// ErrTooManyTimeouts means that the number of services which timed out exceeded
// the threshold, and the deploy should be aborted.
var ErrTooManyTimeouts = errors.New("too many services timed out while restarting")

// ErrTooManyFailures means that a service failed to restart and the deploy
// should be aborted.
var ErrTooManyFailures = errors.New("too many services failed to restart")

// ErrRestartTimeout indicates that a service restart timed out.
type ErrRestartTimeout struct {
	Service string
}

func (e ErrRestartTimeout) Error() string {
	return fmt.Sprintf("restart timed out for service '%s'", e.Service)
}

// ErrRestartFailed indicates that a service restart failed in a manner other
// than timing out.
type ErrRestartFailed struct {
	Service string
	Message string
}

func (e ErrRestartFailed) Error() string {
	return fmt.Sprintf("restart failed for service '%s': %s", e.Service, e.Message)
}
