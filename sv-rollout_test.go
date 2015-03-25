package main

import (
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSvRollout(t *testing.T) {

	Convey("Shuffling services", t, func() {
		defer func() {
			globServices = filepath.Glob
		}()
		unexpected := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
		globServices = func(a string) ([]string, error) {
			var rets []string
			for _, str := range unexpected {
				rets = append(rets, "/etc/service/"+(str))
			}
			return rets, nil
		}
		svcs, err := getServices("borg-shopify-*")

		Convey("Services should be shuffled", func() {
			So(err, ShouldBeNil)
			So(len(svcs), ShouldEqual, len(unexpected))
			for _, str := range unexpected {
				So(svcs, ShouldContain, str)
			}
			// Tiny possibility of random failure.
			So(svcs, ShouldNotResemble, unexpected)
		})
	})

	Convey("Enumerating services", t, func() {
		defer func() {
			globServices = filepath.Glob
		}()
		globServices = func(a string) ([]string, error) {
			return []string{"/etc/service/borg-shopify-test-1", "/etc/service/borg-shopify-test-2"}, nil
		}
		svcs, err := getServices("borg-shopify-*")

		Convey("Should generate corrcect service names", func() {
			So(err, ShouldBeNil)
			So(len(svcs), ShouldEqual, 2)
			So(svcs, ShouldContain, "borg-shopify-test-1")
			So(svcs, ShouldContain, "borg-shopify-test-2")
		})
	})

	Convey("Choosing canaries", t, func() {
		Convey("should not choose any if ratio = 0", func() {
			c, nc := chooseCanaries([]string{"a", "b", "c", "d"}, 0)
			So(len(c), ShouldEqual, 0)
			So(len(nc), ShouldEqual, 4)
		})
		Convey("should choose one if ratio is small", func() {
			c, nc := chooseCanaries([]string{"a", "b", "c", "d", "e", "f", "g"}, 0.001)
			So(len(c), ShouldEqual, 1)
			So(len(nc), ShouldEqual, 6)
		})
		Convey("Should round up in general", func() {
			c, nc := chooseCanaries([]string{"a", "b", "c", "d", "e"}, 0.5)
			So(len(c), ShouldEqual, 3)
			So(len(nc), ShouldEqual, 2)
		})
	})

	Convey("Deciding on permitted timeouts", t, func() {
		Convey("should not allow any if ratio = 0", func() {
			So(permittedTimeouts([]string{"a", "b", "c"}, 0), ShouldEqual, 0)
		})
		Convey("should allow one if ratio is small", func() {
			So(permittedTimeouts([]string{"a", "b", "c", "d", "e", "f"}, 0.001), ShouldEqual, 1)
		})
		Convey("should allow all if ratio is 1", func() {
			So(permittedTimeouts([]string{"a", "b", "c", "d", "e", "f"}, 1), ShouldEqual, 6)
		})
	})
}
