package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// SvRestarter is a simple object that restarts a single runit service.
type SvRestarter struct {
	Service   string
	nServices int
	index     int
	timeout   int
}

var (
	restartGate chan struct{}
	once        sync.Once
)

// Restart shells out to runit to restart the service, and logs messages before
// and after indicating the relevant status.
func (s *SvRestarter) Restart(minInterval time.Duration) error {
	once.Do(func() {
		restartGate = make(chan struct{})
		go func() { restartGate <- struct{}{} }()
	})
	<-restartGate
	go func() {
		time.Sleep(minInterval)
		restartGate <- struct{}{}
	}()

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
	return err
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
}

func (s *SvRestarter) log(message string) {
	printLog(fmt.Sprintf("[%d/%d] (%s) %s", s.index, s.nServices, s.Service, message))
}

func _restartCmd(timeout, service string) ([]byte, error) {
	cmd := exec.Command("/usr/bin/sv", "-w", timeout, "restart", service)
	return cmd.CombinedOutput()
}

// test stubs
var (
	printLog   = log.Println
	restartCmd = _restartCmd
)
