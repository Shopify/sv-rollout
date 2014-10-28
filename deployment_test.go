package main

import (
	"sync"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

var (
	concurrency      = 0
	concurrencyMutex sync.Mutex
)

var (
	quantum = 25 * time.Millisecond
)

func restartWithTiming(svr *SvRestarter) error {
	concurrencyMutex.Lock()
	concurrency++
	concurrencyMutex.Unlock()

	time.Sleep(2 * quantum)

	concurrencyMutex.Lock()
	concurrency--
	concurrencyMutex.Unlock()

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
				So(concurrency, ShouldEqual, 1)
				time.Sleep(2 * quantum)
				So(concurrency, ShouldEqual, 2)
				time.Sleep(2 * quantum)
				So(concurrency, ShouldEqual, 2)
				time.Sleep(2 * quantum)
				So(concurrency, ShouldEqual, 2)
				time.Sleep(2 * quantum)
				So(concurrency, ShouldEqual, 1)
				time.Sleep(2 * quantum)
				So(concurrency, ShouldEqual, 0)

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
				So(concurrency, ShouldEqual, 2)
				time.Sleep(2 * quantum)
				So(concurrency, ShouldEqual, 1)
				time.Sleep(2 * quantum)
				So(concurrency, ShouldEqual, 0)

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
