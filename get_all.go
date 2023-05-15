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

	siteCounts := func(site string) string {
		e, m, te, tm := counter.get(site)
		return fmt.Sprintf("%s@[%d / %d] tot@[%d / %d]", site, e, m, te, tm)
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
		a.VLog(fmt.Sprintf("KILLING - err prop %f (%d/%d)\n", ep, ne, ne+nr))
		counter.kill()
	}

	receive := func() {
		v := <-respOrErrChan
		counter.dec(v.site)
		toLog := fmt.Sprintf("[%d/%d] [%d resps %d errs] receive-get: ", len(resps)+len(errs), len(reqs), len(resps), len(errs))

		if *v == (respOrErr{}) {
			toLog += " empty. "
		} else if v.err != nil {
			toLog += " ERROR. " + siteCounts(v.site)
			errs = append(errs, v.err)
		} else {
			toLog += " success. " + siteCounts(v.site)
			resps = append(resps, v.resp)
		}
		a.VLog(toLog + "\n")
		considerKillIfErrPropHigh()
	}

	send := func(req *GetRequest) {
		if counter.wasKilled() {
			a.VLog("send killed\n")
			respOrErrChan <- &respOrErr{}
			return
		}
		a.VLog("send " + siteCounts(req.Site) + "\n")
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
		a.VLog("was killed - synthesizing errors\n")
		return resps, synthesizeErrors(errs)
	}
	a.VLog("done\n")
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
