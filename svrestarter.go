package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// SvRestarter is a simple object that restarts a single runit service.
type SvRestarter struct {
	Service   string
	nServices int
	index     int
	timeout   int
}

var (
	stdoutLogger = log.New(os.Stdout, "", log.LstdFlags)
	stderrLogger = log.New(os.Stderr, "", log.LstdFlags)
)

// Restart shells out to runit to restart the service, and logs messages before
// and after indicating the relevant status.
func (s *SvRestarter) Restart() error {
	s.log("restarting", false)
	out, err := restartCmd(fmt.Sprintf("%d", s.timeout), s.Service)

	if err != nil {
		if strings.Contains(string(out), "timeout: run: ") {
			err = ErrRestartTimeout{Service: s.Service}
		} else {
			err = ErrRestartFailed{Service: s.Service, Message: string(out)}
		}
	}
	s.notifyResult(err)
	return err
}

func (s *SvRestarter) notifyResult(result error) {
	switch result.(type) {
	case nil:
		s.log("successfully restarted", false)
	case ErrRestartTimeout:
		s.log("did not restart in time", true)
	case ErrRestartFailed:
		s.log("failed to restart", true)
	default:
		panic(result)
	}
}

func (s *SvRestarter) log(message string, toStderr bool) {
	logFunc := stdoutLog
	if toStderr {
		logFunc = stderrLog
	}
	logFunc(fmt.Sprintf("[%d/%d] (%s) %s", s.index, s.nServices, s.Service, message))
}

func _restartCmd(timeout, service string) ([]byte, error) {
	cmd := exec.Command("/usr/bin/sv", "-w", timeout, "restart", service)
	return cmd.CombinedOutput()
}

// test stubs
var (
	stdoutLog  = stdoutLogger.Println
	stderrLog  = stdoutLogger.Println
	restartCmd = _restartCmd
)
