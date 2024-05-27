// Package performance provide basic definitions for building CLI-oriented
// performance tests.
package performance

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/Avalanche-io/counter"
	"github.com/oklog/ulid/v2"
)

// Runner is a performance test runner.
type Runner struct {
	callbackFn     CallbackFn
	duration       time.Duration
	callsPerSecond uint16
	verbose        bool

	errors    sync.Map
	latencies sync.Map
	okCount   *counter.UnsignedCounter
	errCount  *counter.UnsignedCounter

	startedAt time.Time
	summary   *Summary
}

// CallbackFn is the function responsible for executing a unit of work which
// performance is subject to be evaluated.
type CallbackFn func() error

// NewRunner creates a new performance test runner.
func NewRunner(
	duration time.Duration,
	callbackFn CallbackFn,
	callsPerSecond uint16,
	verbose bool,
) *Runner {
	return &Runner{
		callbackFn:     callbackFn,
		duration:       duration,
		callsPerSecond: callsPerSecond,
		verbose:        verbose,

		errors:   sync.Map{},
		okCount:  counter.NewUnsigned(),
		errCount: counter.NewUnsigned(),
	}
}

// Run starts running the performance suite and collecting metrics until the
// given context is canceled, or the duration parameter is reached.
func (p *Runner) Run(ctx context.Context, rampUp *time.Duration) (Summary, error) {
	if p.summary != nil {
		return *p.summary, nil
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	timer := time.NewTimer(p.duration)
	defer timer.Stop()

	timeout := p.duration
	rampUpTkr := time.NewTicker(time.Second)

	if rampUp == nil {
		s := time.Second
		rampUp = &s

		rampUpTkr.Stop()
	}

	delta := float64(p.callsPerSecond) / rampUp.Seconds()
	reqSec := delta

	done := ctx.Done()
	p.startedAt = time.Now()

	fmt.Printf("\r[%s] ", timeout.String())
	defer println()

	var wg sync.WaitGroup

	for {
		select {
		case <-done:
			// stop runner and wait for goroutines to finish
			ticker.Stop()
			wg.Wait()

			return p.buildSummary(), nil
		case <-timer.C:
			// stop runner and wait for goroutines to finish
			ticker.Stop()
			wg.Wait()

			return p.buildSummary(), nil
		case <-ticker.C:
			// time to send the requests for this second
			p.doTick(&wg, reqSec)

			timeout -= time.Second
			if timeout < 0 {
				timeout = 0
			}

			fmt.Printf("\r[%s] ", timeout.String())

			wg.Wait()
		case <-rampUpTkr.C:
			if reqSec >= float64(p.callsPerSecond) {
				rampUpTkr.Stop()

				reqSec = float64(p.callsPerSecond)

				continue
			}

			reqSec += delta
		}
	}
}

func (p *Runner) doTick(wg *sync.WaitGroup, reqSec float64) {
	for i := uint16(0); i < uint16(reqSec); i++ {
		wg.Add(1)

		go func(seq uint16) {
			defer wg.Done()

			// sleep an amount of time so requests are sent (more or less)
			// evenly within the same second.
			time.Sleep(time.Duration(float64(time.Second) / reqSec * float64(seq)))

			startAt := time.Now()
			gErr := p.callbackFn()
			duration := time.Since(startAt)

			defer p.latencies.Store(ulid.Make(), duration)

			if gErr != nil {
				p.debug("F")
				p.errCount.Up()

				loaded, _ := p.errors.LoadOrStore(gErr.Error(), counter.NewUnsigned())
				loaded.(*counter.UnsignedCounter).Up()

				return
			}

			p.debug(".")
			p.okCount.Up()
		}(i)
	}
}

func (p *Runner) buildSummary() Summary {
	total := p.okCount.Get() + p.errCount.Get()
	errPercent := (float64(p.errCount.Get()) * 100) / float64(total)

	if total == 0 {
		return Summary{}
	}

	report := Summary{
		Total:          total,
		Time:           time.Since(p.startedAt),
		Failed:         p.errCount.Get(),
		FailedPercent:  errPercent,
		Success:        p.okCount.Get(),
		SuccessPercent: (float64(p.okCount.Get()) * 100) / float64(total),
		Errors:         map[string]uint64{},
		Latencies:      map[uint8]int64{},
	}

	if errPercent > 0 {
		p.errors.Range(func(key, value interface{}) bool {
			report.Errors[key.(string)] = value.(*counter.UnsignedCounter).Get()

			return true
		})
	}

	samples := make([]int64, 0)

	p.latencies.Range(func(_, obs interface{}) bool {
		samples = append(samples, obs.(time.Duration).Milliseconds())

		return true
	})

	sort.Slice(samples, func(i, j int) bool {
		return samples[i] < samples[j]
	})

	// compute percentiles
	percentiles := []uint8{50, 75, 90, 95, 99}
	for _, percentile := range percentiles {
		index := int(float64(len(samples)) * (float64(percentile) / 100))
		if index == 0 {
			index = 1
		}

		report.Latencies[percentile] = samples[index-1]
	}

	return report
}

func (p *Runner) debug(msg string) {
	if p.verbose {
		print(msg)
	}
}
