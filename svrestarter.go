package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

// SvRestarter is a simple object that restarts a single runit service.
type SvRestarter struct {
	Service   string
	nServices int
	index     int
	timeout   int
	preempt   chan struct{}
}

var (
	stdoutLogger = log.New(os.Stdout, "", log.LstdFlags)
	stderrLogger = log.New(os.Stderr, "", log.LstdFlags)
)

func NewSvRestarter(service string, nServices, index, timeout int) *SvRestarter {
	return &SvRestarter{
		Service:   service,
		nServices: nServices,
		index:     index,
		timeout:   timeout,
		preempt:   make(chan struct{}),
	}
}

// Restart shells out to runit to restart the service, and logs messages before
// and after indicating the relevant status.
func (s *SvRestarter) Restart() error {
	s.log("restarting", false)
	var (
		out                  []byte
		err                  error
		restartDone          = make(chan struct{})
		preemptionAcceptable = make(chan struct{})
	)
	go func() {
		out, err = restartCmd(fmt.Sprintf("%d", s.timeout), s.Service)
		close(restartDone)
	}()

	go func() {
		// We don't want to preempt before actually shelling out, which is kind of
		// hard to hook correctly. Instead we make the kind-of-shady assumption
		// that 200ms is more than enough to fork/exec.
		time.Sleep(200 * time.Millisecond)
		close(preemptionAcceptable)
	}()

	select {
	case <-restartDone:
		if err != nil {
			if strings.Contains(string(out), "timeout: run: ") {
				err = ErrRestartTimeout{Service: s.Service}
			} else {
				err = ErrRestartFailed{Service: s.Service, Message: string(out)}
			}
		}
	case <-s.preempt:
		<-preemptionAcceptable
		err = ErrRestartPreempted{Service: s.Service}
	}

	s.notifyResult(err)
	return err
}

// Preempt instructs an SvRestarter that it need not hang around waiting for a
// restart to complete; that it can just print a message about the service not
// needing to restart successfully for the deploy to be considered a success.
func (s *SvRestarter) Preempt() {
	close(s.preempt)
}

func (s *SvRestarter) notifyResult(result error) {
	switch result.(type) {
	case nil:
		s.log("successfully restarted", false)
	case ErrRestartTimeout:
		s.log("did not restart in time", true)
	case ErrRestartFailed:
		s.log("failed to restart", true)
	case ErrRestartPreempted:
		s.log("was not required to restart in time", true)
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
