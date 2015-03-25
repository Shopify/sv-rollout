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

// ErrRestartPreempted happens when we terminate the deploy early due to a
// sufficient number of services restarting successfully to consider the deploy
// a success even if every remaining service times out.
type ErrRestartPreempted struct {
	Service string
}

func (e ErrRestartPreempted) Error() string {
	return fmt.Sprintf("service '%s' didn't need to restart in time for the deploy to succeed so we stopped watching it", e.Service)
}

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
