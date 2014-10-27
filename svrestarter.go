package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

type ErrRestartTimeout struct {
	Service string
}

func (e ErrRestartTimeout) Error() string {
	return fmt.Sprintf("restart timed out for service '%s'", e.Service)
}

type ErrRestartFailed struct {
	Service string
	Message string
}

func (e ErrRestartFailed) Error() string {
	return fmt.Sprintf("restart failed for service '%s': %s", e.Service, e.Message)
}

type SvRestarter struct {
	Service   string
	nServices int
	index     int
	timeout   int
	results   chan error
}

func (s *SvRestarter) Restart() {
	s.log("restarting")
	out, err := restartCmd(fmt.Sprintf("%d", s.timeout), s.Service)

	if err != nil {
		if strings.Contains(string(out), "timeout: run: ") {
			err = ErrRestartTimeout{Service: s.Service}
		} else {
			err = ErrRestartFailed{Service: s.Service, Message: string(out)}
		}
	}
	s.notifyResult(err)
}

func (s *SvRestarter) notifyResult(result error) {
	switch result.(type) {
	case nil:
		s.log("successfully restarted")
	case ErrRestartTimeout:
		s.log("did not restart in time")
	case ErrRestartFailed:
		s.log("failed to restart")
	default:
		panic(result)
	}
	s.results <- result
}

func (s *SvRestarter) log(message string) {
	printLog(fmt.Sprintf("[%d/%d] (%s) %s", s.index, s.nServices, s.Service, message))
}

func _restartCmd(timeout, service string) ([]byte, error) {
	cmd := exec.Command("sv", "-w", timeout, "restart", service)
	return cmd.CombinedOutput()
}

// test stubs
var (
	printLog   = log.Println
	restartCmd = _restartCmd
)
