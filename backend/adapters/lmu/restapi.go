// restapi.go reads data from LMU's local REST API — things the rF2 shared memory
// does not expose. Today that means VIRTUAL ENERGY (LMU's per-stint regulated
// energy budget), the headline strategic resource in Hypercar/LMDh.
//
// Unlike the shared-memory reader, this is plain HTTP + JSON and therefore builds
// and is fully testable on any OS (see restapi_test.go).
//
// It runs a background poller at a modest rate and caches the latest result, so
// the 10 Hz telemetry read path never blocks on an HTTP request.
//
// Source endpoint: GET {baseURL}/rest/strategy/usage — a map of driver name to a
// per-lap history; each entry has a "ve" field (0.0–1.0 remaining). We keep the
// latest ve per driver. This is the endpoint TinyPedal uses for live energy
// remaining. NOTE: verify the live cadence against a running race — if VE needs
// to update more smoothly than once per lap, /rest/garage/UIScreen/RepairAndRefuel
// (fuelInfo.currentVirtualEnergy / maxVirtualEnergy) is the continuous fallback.
package lmu

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	defaultRestURL   = "http://localhost:6397"
	restPollInterval = 1 * time.Second
	restTimeout      = 800 * time.Millisecond
	usagePath        = "/rest/strategy/usage"
)

// restClient polls the LMU REST API and caches virtual energy per driver.
type restClient struct {
	baseURL string
	http    *http.Client

	mu         sync.RWMutex
	veByDriver map[string]float64 // driver name -> latest virtual energy fraction
	ok         bool               // was the most recent poll successful?

	stop      chan struct{}
	closeOnce sync.Once
	loggedOK  bool
	loggedErr bool
}

func newRESTClient(baseURL string) *restClient {
	return &restClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: restTimeout},
		stop:    make(chan struct{}),
	}
}

// start launches the background poller.
func (c *restClient) start() { go c.loop() }

// close stops the poller. Safe to call more than once.
func (c *restClient) close() { c.closeOnce.Do(func() { close(c.stop) }) }

func (c *restClient) loop() {
	ticker := time.NewTicker(restPollInterval)
	defer ticker.Stop()
	c.pollOnce()
	for {
		select {
		case <-c.stop:
			return
		case <-ticker.C:
			c.pollOnce()
		}
	}
}

// virtualEnergy returns the latest virtual energy fraction (0..1) for a driver,
// or ok=false if the API is unavailable or that driver isn't known.
func (c *restClient) virtualEnergy(driver string) (float64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.ok {
		return 0, false
	}
	ve, ok := c.veByDriver[strings.TrimSpace(driver)]
	return ve, ok
}

func (c *restClient) pollOnce() {
	ve, err := c.fetchUsageVE()
	if err != nil {
		c.mu.Lock()
		c.ok = false
		c.mu.Unlock()
		if !c.loggedErr {
			c.loggedErr, c.loggedOK = true, false
			log.Printf("lmu/rest: virtual energy unavailable (%v) — is LMU's REST API at %s? continuing on shared memory only", err, c.baseURL)
		}
		return
	}
	c.mu.Lock()
	c.veByDriver = ve
	c.ok = true
	c.mu.Unlock()
	if !c.loggedOK {
		c.loggedOK, c.loggedErr = true, false
		log.Printf("lmu/rest: connected to REST API at %s (virtual energy for %d cars)", c.baseURL, len(ve))
	}
}

// usageEntry is one lap's stint-usage record. Only ve is needed today.
type usageEntry struct {
	VE *float64 `json:"ve"`
}

// fetchUsageVE fetches /rest/strategy/usage and returns the latest ve per driver.
func (c *restClient) fetchUsageVE() (map[string]float64, error) {
	resp, err := c.http.Get(c.baseURL + usagePath)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d from %s", resp.StatusCode, usagePath)
	}

	// Shape: { "Driver Name": [ {lap,pit,stint,ve}, ... ], ... }
	var raw map[string][]usageEntry
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode %s: %w", usagePath, err)
	}

	out := make(map[string]float64, len(raw))
	for name, entries := range raw {
		// Take the most recent entry that actually carries a ve value.
		for i := len(entries) - 1; i >= 0; i-- {
			if entries[i].VE != nil {
				out[strings.TrimSpace(name)] = *entries[i].VE
				break
			}
		}
	}
	return out, nil
}
