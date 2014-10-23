package main

import (
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestDeployify(t *testing.T) {

	Convey("Enumerating services", t, func() {
		defer func() {
			globServices = filepath.Glob
		}()
		globServices = func(a string) ([]string, error) {
			return []string{"/etc/sv/borg-shopify-test-1", "/etc/sv/borg-shopify-test-2"}, nil
		}

		Convey("Should generate corrcect service names", func() {
			svcs, err := GetServices("borg-shopify-*")
			So(err, ShouldBeNil)
			So(svcs, ShouldResemble, []string{"borg-shopify-test-1", "borg-shopify-test-2"})
		})
	})
}
