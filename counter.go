package amass

import "sync"

type requestCounter struct {
	allExecuting    int
	allMax          int
	siteToExecuting map[string]int
	siteToMax       map[string]int
	lock            sync.RWMutex
	pendingReceives int
}

func newRequestCounter() *requestCounter {
	return &requestCounter{
		allExecuting:    0,
		allMax:          0,
		siteToExecuting: make(map[string]int),
		siteToMax:       make(map[string]int),
		lock:            sync.RWMutex{},
		pendingReceives: 0,
	}
}

func (c *requestCounter) canSend(site string) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	// If c.allMax == 0, the counter has been killed and no more real work will be done.
	if c.allMax == 0 {
		return true
	}
	if c.siteToExecuting[site] >= c.siteToMax[site] {
		return false
	}
	return c.allExecuting < c.allMax
}

func (c *requestCounter) get(site string) (int, int) {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.siteToExecuting[site], c.siteToMax[site]
}

func (c *requestCounter) inc(site string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.siteToExecuting[site]++
	c.allExecuting++
}

func (c *requestCounter) dec(site string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.siteToExecuting[site]--
	c.allExecuting--
}

func (c *requestCounter) kill() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.allMax = 0
}

func (c *requestCounter) wasKilled() bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.allMax == 0
}

func (c *requestCounter) isActive() bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.allExecuting > 0 && c.allMax > 0
}
