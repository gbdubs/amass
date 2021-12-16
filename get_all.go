package amass

import (
	"errors"
	"fmt"
	"sync"
)

type respOrErr struct {
	site string
	resp *GetResponse
	err  error
}

func (a *Amasser) getAll(reqs []*GetRequest) ([]*GetResponse, error) {
	siteToMax := make(map[string]int)
	for _, req := range reqs {
		siteToMax[req.Site] = req.SiteMaxConcurrentRequests
	}
	counter := newRequestCounter()
	counter.allMax = a.TotalMaxConcurrentRequests
	counter.siteToMax = siteToMax

	respOrErrChan := make(chan *respOrErr)
	resps := make([]*GetResponse, 0)
	errs := make([]error, 0)

	maybeLog := func(s string) {
		if a.Verbose {
			fmt.Printf("  %s\n", s)
		}
	}
	maybeLogSite := func(event string, site string) {
		if a.Verbose {
			e, m := counter.get(site)
			fmt.Printf("  %s - %s@[%d / %d]\n", event, site, e, m)
		}
	}
	send := func(req *GetRequest) {
		if counter.wasKilled() {
			respOrErrChan <- &respOrErr{}
			return
		}
		counter.inc(req.Site)
		maybeLogSite("send", req.Site)
		r := &respOrErr{
			site: req.Site,
		}
		r.resp, r.err = req.Get()
		respOrErrChan <- r
	}
	killLock := sync.RWMutex{}
	considerKillIfErrPropHigh := func() {
		killLock.Lock()
		defer killLock.Unlock()
		if !counter.isActive() {
			return
		}
		ne := len(errs)
		nr := len(resps)
		ep := float64(ne) / float64(ne+nr)
		if ne+nr < 5 || ep < a.AllowedErrorProportion {
			return
		}
		maybeLog(fmt.Sprintf("KILLING - err prop %f (%d/%d)", ep, ne, ne+nr))
		counter.kill()
	}
	receive := func() {
		maybeLog("receive-get")
		v := <-respOrErrChan
		counter.dec(v.site)
		if *v == (respOrErr{}) {
			maybeLog("  received: empty")
		} else if v.err != nil {
			maybeLogSite("  received: err", v.site)
			errs = append(errs, v.err)
		} else {
			maybeLogSite("  received: success", v.site)
			resps = append(resps, v.resp)
		}
		considerKillIfErrPropHigh()
	}
	// Always receive exactly len(reqs) times.
	receivesToDo := len(reqs)
	receivesToDoLock := sync.RWMutex{}
	maybeReceive := func() bool {
		shouldReceive := false
		receivesToDoLock.Lock()
		if receivesToDo > 0 {
			shouldReceive = true
			receivesToDo -= 1
		}
		receivesToDoLock.Unlock()
		if shouldReceive {
			receive()
			return true
		} else {
			maybeLog("maybe receive - no")
			return false
		}
	}
	// Always send exactly n times, even if killed.
	sendRequests := func() {
		for _, req := range reqs {
			r := req
			for true {
				if counter.canSend(r.Site) {
					break
				}
				maybeReceive()
			}
			go send(r)
		}
	}

	go sendRequests()

	for maybeReceive() {
	}

	if counter.wasKilled() {
		maybeLog("was killed - synthesizing errors")
		return resps, synthesizeErrors(errs)
	}
	maybeLog("done")
	return resps, nil
}

func synthesizeErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[1]
	}
	s := "Encountered %d errors:\n"
	for i, err := range errs {
		s += fmt.Sprintf("  %d:  %v\n", i, err)
	}
	return errors.New(s)
}
