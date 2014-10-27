/*
deployify is a utility to restart multiple runit services concurrently. It
supports canaries and has configurable tolerance for timeouts.
*/
package main

import (
	"flag"
	"log"
	"os"
	"path"
	"path/filepath"
)

const (
	svdir = "/etc/sv"
)

type config struct {
	CanaryRatio            float64
	CanaryTimeoutTolerance float64
	ChunkRatio             float64
	TimeoutTolerance       float64
	Timeout                int
}

func (c config) AssertValid() {
	if c.ChunkRatio < c.CanaryRatio {
		log.Fatal("-chunk-ratio must be >= -canary-ratio. This is not an inherent limitation, feel free to add code to handle this case.")
	}
}

func main() {
	var (
		canaryRatio            = flag.Float64("canary-ratio", 0.001, "canary nodes are restarted first. If they fail, the deploy is failed. Rounded up to the nearest node, unless set to zero")
		canaryTimeoutTolerance = flag.Float64("canary-timeout-tolerance", 0, "ratio of canary nodes that are permitted to time out without causing the deploy to fail")
		chunkRatio             = flag.Float64("chunk-ratio", 0.2, "after canary nodes, ratio of remaining nodes permitted to restart concurrently")
		timeoutTolerance       = flag.Float64("timeout-tolerance", 0, "ratio of total nodes whose restarts may time out and still consider the deploy a success")
		timeout                = flag.Int("timeout", 90, "number of seconds to wait for a service to restart before considering it timed out and moving on")
		pattern                = flag.String("pattern", "", "(required) glob pattern to match /etc/sv entries (e.g. \"borg-shopify-*\")")
	)
	flag.Parse()

	config := config{
		CanaryRatio:            *canaryRatio,
		CanaryTimeoutTolerance: *canaryTimeoutTolerance,
		ChunkRatio:             *chunkRatio,
		TimeoutTolerance:       *timeoutTolerance,
		Timeout:                *timeout,
	}
	config.AssertValid()
	if *pattern == "" {
		log.Fatal("-pattern must be provided")
	}

	os.Exit(run(*pattern, config))
}

func run(servicePattern string, c config) int {
	services, err := getServices(servicePattern)
	if err != nil {
		log.Fatal(err)
	}

	d := NewDeployment(services, c)
	if err := d.Run(); err != nil {
		return 1
	}
	return 0
}

func getServices(pattern string) (services []string, err error) {
	var fullpaths []string
	fullpaths, err = globServices(svdir + "/" + pattern)
	if err != nil {
		return nil, err
	}
	for _, p := range fullpaths {
		services = append(services, path.Base(p))
	}
	return
}

var globServices = filepath.Glob
