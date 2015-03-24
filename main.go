/*
sv-rollout is a utility to restart multiple runit services concurrently. It
supports canaries and has configurable tolerance for timeouts.
*/
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
)

const (
	svdir = "/etc/service"
)

type config struct {
	CanaryRatio            float64
	CanaryTimeoutTolerance float64
	ChunkRatio             float64
	TimeoutTolerance       float64
	Timeout                int
	OnComplete             string
}

func init() {
	log.SetOutput(os.Stdout)
}

// Verbose controls whether extra information is printed as choices are made.
// It's set by the `-verbose` CLI flag.
var Verbose bool

func (c config) AssertValid(pattern string) {
	msg := ""
	if pattern == "" {
		msg = "-pattern must be provided"
	}
	if c.ChunkRatio < c.CanaryRatio {
		msg = "-chunk-ratio must be >= -canary-ratio. This is not an inherent limitation, feel free to add code to handle this case."
	}
	if msg != "" {
		fmt.Println(msg)
		flag.Usage()
		os.Exit(1)
	}
}

func init() {
	oldUsage := flag.Usage
	flag.Usage = func() {
		oldUsage()
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintln(os.Stderr, "  # Restart one service first. Restart everything else once it succeeds. No timeouts allowed, wait up to 5 minutes for restarts.")
		fmt.Fprintln(os.Stderr, "  "+os.Args[0]+" -canary-ratio 0.0001 -chunk-ratio 1 -timeout 300 -pattern 'borg-*'")
		fmt.Fprintf(os.Stderr, "%s", "  # Restart 10% of services first, allowing up to 50% of those to time out. Then, restart all other services, 30% at a time, allowing up to 70% to time out.\n")
		fmt.Fprintln(os.Stderr, "  "+os.Args[0]+" -canary-ratio 0.1 -chunk-ratio 0.3 -canary-timeout-tolerance 0.5 -timeout-tolerance 0.7 -timeout 300 -pattern 'borg-*'")
	}
}

func main() {
	var (
		canaryRatio            = flag.Float64("canary-ratio", 0.001, "canary nodes are restarted first. If they fail, the deploy is failed. Rounded up to the nearest node, unless set to zero")
		canaryTimeoutTolerance = flag.Float64("canary-timeout-tolerance", 0, "ratio of canary nodes that are permitted to time out without causing the deploy to fail")
		chunkRatio             = flag.Float64("chunk-ratio", 0.2, "after canary nodes, ratio of remaining nodes permitted to restart concurrently")
		timeoutTolerance       = flag.Float64("timeout-tolerance", 0, "ratio of total nodes whose restarts may time out and still consider the deploy a success")
		timeout                = flag.Int("timeout", 90, "number of seconds to wait for a service to restart before considering it timed out and moving on")
		pattern                = flag.String("pattern", "", "(required) glob pattern to match /etc/service entries (e.g. \"borg-shopify-*\")")
		onComplete             = flag.String("oncomplete", "", "command to execute when the deploy finishes (regardless of success)")
		verbose                = flag.Bool("verbose", false, "print more information about what's going on")
	)

	flag.Parse()
	config := config{
		CanaryRatio:            *canaryRatio,
		CanaryTimeoutTolerance: *canaryTimeoutTolerance,
		ChunkRatio:             *chunkRatio,
		TimeoutTolerance:       *timeoutTolerance,
		Timeout:                *timeout,
		OnComplete:             *onComplete,
	}
	config.AssertValid(*pattern)

	Verbose = *verbose

	if Verbose {
		log.Printf("[debug] initializing with pattern=%s, config %#v", *pattern, config)
	}

	os.Exit(run(*pattern, config))
}

func run(servicePattern string, c config) int {
	services, err := getServices(servicePattern)
	if err != nil {
		log.Fatal(err)
	}

	defer runCompletionHandler(c.OnComplete)

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

func runCompletionHandler(command string) {
	if command == "" {
		return
	}
	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("completion handler failed:", err)
		return
	}
	log.Println("completion handler:", command)
	log.Println(string(output))
}

var globServices = filepath.Glob
