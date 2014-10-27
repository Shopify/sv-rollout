package main

import (
	"math"
)

type Deployment struct {
	numServices int

	canaryServices     []string
	postCanaryServices []string

	canaryTimeoutsPermitted int
	totalTimeoutsPermitted  int

	postCanaryConcurrency int

	timeout int

	index int
}

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

	return &d
}

func (d *Deployment) Run() (err error) {
	if err = d.restartCanaries(); err != nil {
		return
	}
	return
}

func (d *Deployment) restartCanaries() (err error) {
	results := make(chan error)
	for _, svc := range d.canaryServices {
		d.beginRestart(results, svc)
	}
	return
}

func (d *Deployment) beginRestart(results chan error, service string) {
	d.index++
	svr := SvRestarter{
		Service:   service,
		nServices: d.numServices,
		index:     d.index,
		timeout:   d.timeout,
		results:   results,
	}
	go svr.Restart()
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
