package main

import (
	"sync/atomic"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

var (
	concurrency int32
)

var (
	quantum = 25 * time.Millisecond
)

func restartWithTiming(svr *SvRestarter) error {
	atomic.AddInt32(&concurrency, 1)
	time.Sleep(2 * quantum)
	atomic.AddInt32(&concurrency, -1)

	return nil
}

func alwaysPass(svr *SvRestarter) error {
	return nil
}

func alwaysTimeout(svr *SvRestarter) error {
	return ErrRestartTimeout{Service: svr.Service}
}

func alwaysFail(svr *SvRestarter) error {
	return ErrRestartFailed{Service: svr.Service, Message: "failed"}
}

func timeoutOneService(svr *SvRestarter) error {
	if svr.Service == "b" {
		return alwaysTimeout(svr)
	}
	return alwaysPass(svr)
}

func failOneService(svr *SvRestarter) error {
	if svr.Service == "b" {
		return alwaysFail(svr)
	}
	return alwaysPass(svr)
}

func TestDeployment(t *testing.T) {

	config := config{
		CanaryRatio:            0.0001,
		CanaryTimeoutTolerance: 0,
		ChunkRatio:             0.2,
		TimeoutTolerance:       0,
		Timeout:                1,
	}

	Convey("Running a deployment", t, func() {

		Convey("with preemption", func() {
			config.CanaryRatio = 0
			config.TimeoutTolerance = 0.61
			config.ChunkRatio = 0.001
			restartSvr = _restartSvr

			restartCmd = func(t, s string, a chan struct{}) ([]byte, error) {
				close(a)
				time.Sleep(250 * time.Millisecond)
				return nil, nil
			}
			var outLogs []string
			var errLogs []string
			stdoutLog = func(a ...interface{}) { outLogs = append(outLogs, a[0].(string)) }
			stderrLog = func(a ...interface{}) { errLogs = append(errLogs, a[0].(string)) }

			depl := NewDeployment([]string{"a", "b", "c", "d", "e"}, config)
			t1 := time.Now()
			err := depl.Run()
			t2 := time.Since(t1)

			Convey("succeeds, faster than without preemption", func() {
				// tolerance = 60%, therefore only need to wait on 2/5.
				// 2 * 250ms = 500ms, but will take usually ~510-520ms.
				So(t2, ShouldBeBetween, 500*time.Millisecond, 599*time.Millisecond)
				So(err, ShouldBeNil)

				So(outLogs, ShouldResemble, []string{
					"[1/5] (a) restarting",
					"[1/5] (a) successfully restarted",
					"[2/5] (b) restarting",
					"[2/5] (b) successfully restarted",
					"[3/5] (c) restarting",
					"[4/5] (d) restarting",
					"[5/5] (e) restarting",
				})
				So(errLogs, ShouldResemble, []string{
					"[3/5] (c) was not required to restart in time",
					"[4/5] (d) was not required to restart in time",
					"[5/5] (e) was not required to restart in time",
				})

			})
		})

		Convey("with no canaries and 50% timeouts allowed", func() {
			config.CanaryRatio = 0
			config.TimeoutTolerance = 0.5
			config.ChunkRatio = 0.001
			Convey("succeeds when everything restarts successfully", func() {
				restartSvr = alwaysPass
				depl := NewDeployment([]string{"a", "b", "c"}, config)
				err := depl.Run()
				So(err, ShouldBeNil)
			})
			Convey("fails when everything times out", func() {
				restartSvr = alwaysTimeout
				depl := NewDeployment([]string{"a", "b", "c"}, config)
				err := depl.Run()
				So(err, ShouldEqual, ErrTooManyTimeouts)
			})
			Convey("fails when everything fails", func() {
				restartSvr = alwaysFail
				depl := NewDeployment([]string{"a", "b", "c"}, config)
				err := depl.Run()
				So(err, ShouldEqual, ErrTooManyFailures)
			})
			Convey("fails when only one service fails", func() {
				restartSvr = failOneService
				depl := NewDeployment([]string{"a", "b", "c"}, config)
				err := depl.Run()
				So(err, ShouldEqual, ErrTooManyFailures)
			})
			Convey("succeeds when only one service times out", func() {
				restartSvr = timeoutOneService
				depl := NewDeployment([]string{"a", "b", "c"}, config)
				err := depl.Run()
				So(err, ShouldBeNil)
			})
		})

		Convey("with one canary followed by 100%", func() {
			config.CanaryRatio = 0.001
			config.ChunkRatio = 1
			config.TimeoutTolerance = 0.5
			Convey("Fails when the canary times out", func() {
				depl := NewDeployment([]string{"a", "b", "c"}, config)
				restartSvr = func(svr *SvRestarter) error {
					if svr.Service == depl.canaryServices[0] {
						return alwaysTimeout(svr)
					}
					return nil
				}
				err := depl.Run()
				So(err, ShouldEqual, ErrTooManyTimeouts)
			})
			Convey("Succeeds when one non-canary service times out", func() {
				depl := NewDeployment([]string{"a", "b", "c"}, config)
				restartSvr = func(svr *SvRestarter) error {
					if svr.Service == depl.postCanaryServices[0] {
						return alwaysTimeout(svr)
					}
					return nil
				}
				err := depl.Run()
				So(err, ShouldBeNil)
			})
			Convey("Fails when the canary fails", func() {
				depl := NewDeployment([]string{"a", "b", "c"}, config)
				restartSvr = func(svr *SvRestarter) error {
					if svr.Service == depl.canaryServices[0] {
						return alwaysFail(svr)
					}
					return nil
				}
				err := depl.Run()
				So(err, ShouldEqual, ErrTooManyFailures)
			})
			Convey("Succeeds when one non-canary service fails", func() {
				depl := NewDeployment([]string{"a", "b", "c"}, config)
				restartSvr = func(svr *SvRestarter) error {
					if svr.Service == depl.postCanaryServices[0] {
						return alwaysFail(svr)
					}
					return nil
				}
				err := depl.Run()
				So(err, ShouldEqual, ErrTooManyFailures)
			})
		})

	})

	Convey("Running a deployment with canary-ratio 0.001 and chunk-ratio 0.25", t, func() {
		config.CanaryRatio = 0.001
		config.ChunkRatio = 0.25
		Convey("on 8 nodes", func() {
			depl := NewDeployment([]string{"a", "b", "c", "d", "e", "f", "g", "h"}, config)
			restartSvr = restartWithTiming

			Convey("should restart the canary alone, then two nodes concurrently", func() {
				ch := make(chan error)
				go func() {
					ch <- depl.Run()
				}()
				time.Sleep(quantum) // put us out of phase with the sleeps in the restart code
				So(atomic.LoadInt32(&concurrency), ShouldEqual, 1)
				time.Sleep(2 * quantum)
				So(atomic.LoadInt32(&concurrency), ShouldEqual, 2)
				time.Sleep(2 * quantum)
				So(atomic.LoadInt32(&concurrency), ShouldEqual, 2)
				time.Sleep(2 * quantum)
				So(atomic.LoadInt32(&concurrency), ShouldEqual, 2)
				time.Sleep(2 * quantum)
				So(atomic.LoadInt32(&concurrency), ShouldEqual, 1)
				time.Sleep(2 * quantum)
				So(atomic.LoadInt32(&concurrency), ShouldEqual, 0)

				var err error
				select {
				case err = <-ch:
					So(err, ShouldBeNil)
				default:
					// should have been done
					So(nil, ShouldNotBeNil)
				}
			})
		})
	})

	Convey("Running a deployment with canary-ratio 0 and chunk-ratio 0.5", t, func() {
		config.CanaryRatio = 0
		config.ChunkRatio = 0.5
		Convey("on 3 nodes", func() {
			depl := NewDeployment([]string{"a", "b", "c"}, config)
			restartSvr = restartWithTiming

			Convey("should restart two nodes, then the last one", func() {
				ch := make(chan error)
				go func() {
					ch <- depl.Run()
				}()
				time.Sleep(quantum) // put us out of phase with the sleeps in the restart code
				So(atomic.LoadInt32(&concurrency), ShouldEqual, 2)
				time.Sleep(2 * quantum)
				So(atomic.LoadInt32(&concurrency), ShouldEqual, 1)
				time.Sleep(2 * quantum)
				So(atomic.LoadInt32(&concurrency), ShouldEqual, 0)

				var err error
				select {
				case err = <-ch:
					So(err, ShouldBeNil)
				default:
					// should have been done
					So(nil, ShouldNotBeNil)
				}
			})
		})

	})

}
