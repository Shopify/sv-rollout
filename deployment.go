package main

import (
	"log"
	"math"
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

	servicesToRestart chan string
	results           chan error
}

// NewDeployment initializes a Deployment object with a list of services
// (entries in /etc/sv typically) and a config object.
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

	d.servicesToRestart = make(chan string, 8192)
	d.results = make(chan error, 32)

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
	if err = d.restartCanaries(); err != nil {
		return
	}
	delta := d.postCanaryConcurrency - len(d.canaryServices)
	d.startWorkers(delta)
	return d.restartRemaining()
}

func (d *Deployment) startWorkers(n int) {
	for i := 0; i < n; i++ {
		go d.startWorker()
	}
}

func (d *Deployment) startWorker() {
	for service := range d.servicesToRestart {
		d.results <- d.restart(service)
	}
}

func (d *Deployment) restartCanaries() (err error) {
	d.currentFailuresPermitted = 0
	d.currentTimeoutsPermitted = d.canaryTimeoutsPermitted
	if len(d.canaryServices) == 0 {
		return nil
	}
	for _, svc := range d.canaryServices {
		d.servicesToRestart <- svc
	}

	for result := range d.results {
		switch result.(type) {
		case nil:
			d.successesSoFar++
		case ErrRestartFailed:
			d.failuresSoFar++
		case ErrRestartTimeout:
			d.timeoutsSoFar++
		default:
			panic(result)
		}
		if d.tooManyFailures() {
			return ErrTooManyFailures
		}
		if d.tooManyTimeouts() {
			return ErrTooManyTimeouts
		}
		if d.canarySuccessOK() {
			return nil
		}
	}

	panic("unreachable")
}

func (d *Deployment) restartRemaining() (err error) {
	d.currentFailuresPermitted = 0
	d.currentTimeoutsPermitted = d.totalTimeoutsPermitted

	for _, svc := range d.postCanaryServices {
		d.servicesToRestart <- svc
	}

	for result := range d.results {
		switch result.(type) {
		case nil:
			d.successesSoFar++
		case ErrRestartFailed:
			d.failuresSoFar++
		case ErrRestartTimeout:
			d.timeoutsSoFar++
		default:
			panic(result)
		}
		if d.tooManyFailures() {
			return ErrTooManyFailures
		}
		if d.tooManyTimeouts() {
			return ErrTooManyTimeouts
		}
		if d.allComplete() {
			return nil
		}
	}

	panic("unreachable")
}

func (d *Deployment) tooManyFailures() bool {
	return d.failuresSoFar > d.currentFailuresPermitted
}

func (d *Deployment) tooManyTimeouts() bool {
	return d.timeoutsSoFar > d.currentTimeoutsPermitted
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

func (d *Deployment) restart(service string) error {
	d.index++
	svr := SvRestarter{
		Service:   service,
		nServices: d.numServices,
		index:     d.index,
		timeout:   d.timeout,
	}
	return restartSvr(&svr)
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
