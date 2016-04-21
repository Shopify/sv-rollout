package main

import (
	"log"
	"math"
	"os"
)

// Deployment orchestrates the concurrent restarting of all the indicated
// services, ultimately returning an error indicating whether it was
// successful.
type Deployment struct {
	numServices int

	canaryServices     []string
	postCanaryServices []string

	canaryTimeoutsPermitted int
	totalTimeoutsPermitted  int

	postCanaryConcurrency int

	timeout int
	index   int

	currentFailuresPermitted int
	currentTimeoutsPermitted int

	successesSoFar int
	timeoutsSoFar  int
	failuresSoFar  int

	svrs []*SvRestarter

	toRestart chan *SvRestarter
	results   chan error
}

// NewDeployment initializes a Deployment object with a list of services
// (entries in /etc/service typically) and a config object.
func NewDeployment(services []string, config config) *Deployment {
	var d Deployment
	d.numServices = len(services)

	d.canaryServices, d.postCanaryServices = chooseCanaries(services, config.CanaryRatio)
	d.canaryTimeoutsPermitted = permittedTimeouts(d.canaryServices, config.CanaryTimeoutTolerance)

	ctp := d.canaryTimeoutsPermitted
	ttp := ctp + permittedTimeouts(d.postCanaryServices, config.TimeoutTolerance)
	d.totalTimeoutsPermitted = ttp

	d.timeout = config.Timeout

	d.postCanaryConcurrency = ceilRatio(d.postCanaryServices, config.ChunkRatio)

	d.toRestart = make(chan *SvRestarter, 8192)
	d.results = make(chan error, 1024)

	if Verbose {
		log.Printf("[debug] chose canaries: %v", d.canaryServices)
		log.Printf("[debug] canaries permitted to time out: %d", d.canaryTimeoutsPermitted)
		log.Printf("[debug] total timeouts permitted: %d", d.totalTimeoutsPermitted)
		log.Printf("[debug] concurrency after canary phase: %d", d.postCanaryConcurrency)
	}

	return &d
}

// Run does all the actual grunt work of concurrently restarting the services.
// It first attempts to restart the services chosen as canaries. If a
// sufficient number of them pass, it will move on to restarting the rest of
// the services with concurrency as indicated by ChunkRatio.
func (d *Deployment) Run() (err error) {
	d.startWorkers(len(d.canaryServices))
	if err = d.restartServices(d.canaryServices, 0, d.canaryTimeoutsPermitted, d.canarySuccessOK); err != nil {
		return
	}
	delta := d.postCanaryConcurrency - len(d.canaryServices)
	d.startWorkers(delta)
	return d.restartServices(d.postCanaryServices, 0, d.totalTimeoutsPermitted, d.allComplete)
}

func (d *Deployment) startWorkers(n int) {
	for i := 0; i < n; i++ {
		go d.startWorker()
	}
}

func (d *Deployment) startWorker() {
	for svr := range d.toRestart {
		d.results <- restartSvr(svr)
	}
}

func (d *Deployment) restartServices(services []string, failuresPermitted, timeoutsPermitted int, done func() bool) (err error) {
	d.currentFailuresPermitted = failuresPermitted
	d.currentTimeoutsPermitted = timeoutsPermitted

	if len(services) == 0 {
		return nil
	}

	// Don't restart services when a lock is present
	if _, err := os.Stat("/var/lock/dont-sv-rollout"); err == nil {
		log.Println("/var/lock/dont-sv-rollout present, not restarting services")
		return nil
	}

	for _, svc := range services {
		d.index++
		svr := NewSvRestarter(svc, d.numServices, d.index, d.timeout)
		d.svrs = append(d.svrs, svr)
		d.toRestart <- svr
	}

	remaining := len(services) // number of services yet to be processed.
	for result := range d.results {
		switch result.(type) {
		case nil:
			d.successesSoFar++
		case ErrRestartFailed:
			if err = d.incrementFailures(); err != nil {
				return
			}
		case ErrRestartTimeout:
			if err = d.incrementTimeouts(); err != nil {
				return
			}
		case ErrRestartPreempted:
			// no need to handle the error here because we pre-verified that it's ok
			// before preempting the svr
			_ = d.incrementTimeouts()
		default:
			panic(result)
		}
		if done() {
			return nil
		}

		if d.currentTimeoutsPermitted >= remaining {
			for _, svr := range d.svrs {
				svr.Preempt()
			}
		}
		remaining--
	}

	panic("unreachable")
}

func (d *Deployment) incrementFailures() error {
	d.failuresSoFar++
	if d.failuresSoFar > d.currentFailuresPermitted {
		return ErrTooManyFailures
	}
	return nil
}

func (d *Deployment) incrementTimeouts() error {
	d.timeoutsSoFar++
	if d.timeoutsSoFar > d.currentTimeoutsPermitted {
		return ErrTooManyTimeouts
	}
	return nil
}

func (d *Deployment) canarySuccessOK() bool {
	canaries := len(d.canaryServices)
	mustPass := canaries - d.canaryTimeoutsPermitted
	return d.successesSoFar >= mustPass
}

func (d *Deployment) allComplete() bool {
	done := d.successesSoFar + d.timeoutsSoFar + d.failuresSoFar
	return done == d.numServices
}

func chooseCanaries(services []string, ratio float64) (canaries []string, nonCanaries []string) {
	nCanary := int(math.Ceil(ratio * float64(len(services))))
	for index, service := range services {
		if index < nCanary {
			canaries = append(canaries, service)
		} else {
			nonCanaries = append(nonCanaries, service)
		}
	}
	return
}

func permittedTimeouts(services []string, tolerance float64) int {
	return ceilRatio(services, tolerance)
}

func ceilRatio(coll []string, ratio float64) int {
	return int(math.Ceil(ratio * float64(len(coll))))
}

// stubbed in tests
var (
	restartSvr = _restartSvr
)

func _restartSvr(svr *SvRestarter) error {
	return svr.Restart()
}
