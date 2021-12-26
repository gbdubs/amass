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
			e, m, te, tm := counter.get(site)
			fmt.Printf("  %s - %s@[%d / %d] tot@[%d / %d]\n", event, site, e, m, te, tm)
		}
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

	send := func(req *GetRequest) {
		if counter.wasKilled() {
			maybeLog("send killed")
			respOrErrChan <- &respOrErr{}
			return
		}
		maybeLogSite("send", req.Site)
		r := &respOrErr{
			site: req.Site,
		}
		r.resp, r.err = req.Get()
		respOrErrChan <- r
	}

	sentReqs := make([]bool, len(reqs))
	sentReqsLock := sync.RWMutex{}
	tryToSendAllUnsentRequests := func() {
		sentReqsLock.Lock()
		for i, req := range reqs {
			if sentReqs[i] {
				continue
			}
			r := req
			if counter.incIfCanSend(r.Site) {
				sentReqs[i] = true
				go send(r)
			}
		}
		sentReqsLock.Unlock()
	}

	tryToSendAllUnsentRequests()

	for i := 0; i < len(reqs); i++ {
		receive()
		tryToSendAllUnsentRequests()
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
