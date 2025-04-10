package limiter

import (
	"fmt"
	"github.com/behavioral-ai/collective/eventing"
	"github.com/behavioral-ai/collective/exchange"
	"github.com/behavioral-ai/collective/timeseries"
	"github.com/behavioral-ai/core/access"
	"github.com/behavioral-ai/core/httpx"
	"github.com/behavioral-ai/core/messaging"
	"github.com/behavioral-ai/traffic/config"
	"golang.org/x/time/rate"
	"net/http"
	"time"
)

// Namespace ID Namespace Specific String
// NID + NSS
// NamespaceName
const (
	NamespaceName   = "resiliency:agent/behavioral-ai/traffic/rate-limiting"
	offPeakDuration = time.Minute * 5
	peakDuration    = time.Minute * 2
	defaultLimit    = rate.Limit(50)
	defaultBurst    = 10
	loadSize        = 200
)

type agentT struct {
	running   bool
	enabled   bool
	limiter   *rate.Limiter
	events    *list
	threshold timeseries.Percentile
	actual    timeseries.Percentile

	ticker     *messaging.Ticker
	master     *messaging.Channel
	emissary   *messaging.Channel
	handler    eventing.Agent
	dispatcher messaging.Dispatcher
}

// New - create a new agent
func init() {
	a := newAgent(eventing.Handler)
	exchange.Register(a)
}

func newAgent(handler eventing.Agent) *agentT {
	a := new(agentT)
	a.limiter = rate.NewLimiter(defaultLimit, defaultBurst)
	a.handler = handler
	a.events = newList()

	a.ticker = messaging.NewTicker(messaging.ChannelEmissary, peakDuration)
	a.master = messaging.NewMasterChannel()
	a.emissary = messaging.NewEmissaryChannel()
	return a
}

// String - identity
func (a *agentT) String() string { return a.Uri() }

// Uri - agent identifier
func (a *agentT) Uri() string { return NamespaceName }

// Message - message the agent
func (a *agentT) Message(m *messaging.Message) {
	if m == nil {
		return
	}
	if !a.running {
		if m.Event() == messaging.ConfigEvent {
			a.configure(m)
			return
		}
		if m.Event() == messaging.StartupEvent {
			a.run()
			a.running = true
			return
		}
		return
	}
	if m.Event() == messaging.ShutdownEvent {
		a.running = false
	}
	switch m.Channel() {
	case messaging.ChannelMaster:
		a.master.C <- m
	case messaging.ChannelControl:
		a.master.C <- m
	default:
		fmt.Printf("limiter - invalid channel %v\n", m)
	}
}

// Run - run the agent
func (a *agentT) run() {
	go emissaryAttend(a)
	go masterAttend(a, timeseries.Functions)
}

// Link - chainable exchange
func (a *agentT) Link(next httpx.Exchange) httpx.Exchange {
	return func(req *http.Request) (resp *http.Response, err error) {
		start := time.Now().UTC()
		if !a.limiter.Allow() {
			h := make(http.Header)
			h.Add(access.XRateLimit, fmt.Sprintf("%v", a.limiter.Limit()))
			h.Add(access.XRateBurst, fmt.Sprintf("%v", a.limiter.Burst()))
			return &http.Response{StatusCode: http.StatusTooManyRequests, Header: h}, nil
		}

		if next != nil {
			resp, err = next(req)
		} else {
			resp = &http.Response{StatusCode: http.StatusOK}
		}
		if a.enabled {
			a.events.Enqueue(&event{Start: start, Duration: time.Since(start), StatusCode: resp.StatusCode})
		}
		return
	}
}

func (a *agentT) dispatch(channel any, event string) {
	if a.dispatcher != nil {
		a.dispatcher.Dispatch(a, channel, event)
	}
}

func (a *agentT) emissaryShutdown() {
	a.ticker.Stop()
	a.emissary.Close()
}

func (a *agentT) masterShutdown() {
	a.master.Close()
}

func (a *agentT) reviseTicker(cnt int) {
	var (
		newDuration time.Duration
	)

	if cnt == loadSize {
		return
	}
	if cnt > 2*loadSize {
		newDuration = peakDuration
	} else {
		if cnt < loadSize/2 {
			newDuration = offPeakDuration
		}
	}
	if newDuration != 0 {
		a.ticker.Start(newDuration)
	}
}

func (a *agentT) configure(m *messaging.Message) {
	switch m.ContentType() {
	case messaging.ContentTypeMap:
		var ok bool
		if a.threshold, ok = config.Percentile(a, m); !ok {
			return
		}
		a.actual.Score = a.threshold.Score
	case messaging.ContentTypeDispatcher:
		if dispatcher, ok := messaging.DispatcherContent(m); ok {
			a.dispatcher = dispatcher
		}
	}
	messaging.Reply(m, messaging.StatusOK(), a.Uri())
}
