package main

import (
	"os/exec"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSvRestarter(t *testing.T) {
	var logs []string
	printLog = func(a ...interface{}) { logs = append(logs, a[0].(string)) }

	Convey("When a service restarts successfully under SvRestarter", t, func() {
		logs = []string{}
		restartCmd = func(t, s string) ([]byte, error) {
			return nil, nil
		}
		svr := SvRestarter{Service: "/etc/service/my-test-service", nServices: 3, index: 2, timeout: 1}
		Convey("the results channel should get a nil and a success message should be printed", func() {
			err := svr.Restart(0)
			So(err, ShouldBeNil)
			So(logs, ShouldResemble, []string{
				"[2/3] (/etc/service/my-test-service) restarting",
				"[2/3] (/etc/service/my-test-service) successfully restarted",
			})
		})
	})

	Convey("When a service fails to restart under SvRestarter", t, func() {
		logs = []string{}
		restartCmd = func(t, s string) ([]byte, error) {
			return exec.Command("sh", "-c", "echo failed && false").Output()
		}
		svr := SvRestarter{Service: "/etc/service/my-test-service", nServices: 3, index: 2, timeout: 1}
		Convey("the results channel should get an error and a message should be printed", func() {
			err := svr.Restart(0)
			So(err.(ErrRestartFailed), ShouldNotBeNil)
			So(logs, ShouldResemble, []string{
				"[2/3] (/etc/service/my-test-service) restarting",
				"[2/3] (/etc/service/my-test-service) failed to restart",
			})
		})
	})

	Convey("When a service times out under SvRestarter", t, func() {
		logs = []string{}
		restartCmd = func(t, s string) ([]byte, error) {
			return exec.Command("sh", "-c", "echo 'timeout: run: stuff' && false").Output()
		}
		svr := SvRestarter{Service: "/etc/service/my-test-service", nServices: 3, index: 2, timeout: 1}
		Convey("the results channel should get an error and a message should be printed", func() {
			err := svr.Restart(0)
			So(err.(ErrRestartTimeout), ShouldNotBeNil)
			So(logs, ShouldResemble, []string{
				"[2/3] (/etc/service/my-test-service) restarting",
				"[2/3] (/etc/service/my-test-service) did not restart in time",
			})
		})
	})

}
