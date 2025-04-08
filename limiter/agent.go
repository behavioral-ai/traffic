package limiter

import (
	"fmt"
	"github.com/behavioral-ai/collective/content"
	"github.com/behavioral-ai/collective/eventing"
	"github.com/behavioral-ai/collective/exchange"
	"github.com/behavioral-ai/collective/timeseries"
	"github.com/behavioral-ai/core/access"
	"github.com/behavioral-ai/core/httpx"
	"github.com/behavioral-ai/core/messaging"
	"github.com/behavioral-ai/traffic/metrics"

	"golang.org/x/time/rate"
	"net/http"
	"time"
)

// Namespace ID Namespace Specific String
// NID + NSS
// NamespaceName
const (
	NamespaceName = "resiliency:agent/behavioral-ai/traffic/rate-limiting"
	minDuration   = time.Second * 10
	maxDuration   = time.Second * 15
	defaultLimit  = rate.Limit(50)
	defaultBurst  = 10
)

type agentT struct {
	running bool
	limiter *rate.Limiter

	ticker     *messaging.Ticker
	emissary   *messaging.Channel
	master     *messaging.Channel
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

	a.ticker = messaging.NewTicker(messaging.Emissary, maxDuration)
	a.emissary = messaging.NewEmissaryChannel()
	a.master = messaging.NewMasterChannel()
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
	case messaging.Emissary:
		a.emissary.C <- m
	case messaging.Master:
		a.master.C <- m
	case messaging.Control:
		if m.Event() == metrics.Event {
			a.master.C <- m
		} else {
			//a.emissary.C <- m
			//a.master.C <- m
		}
	default:
		a.emissary.C <- m
	}
}

// Run - run the agent
func (a *agentT) run() {
	go masterAttend(a, timeseries.Functions)
	go emissaryAttend(a, content.Resolver, nil)
}

// Link - chainable exchange
func (a *agentT) Link(next httpx.Exchange) httpx.Exchange {
	return func(req *http.Request) (resp *http.Response, err error) {
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
		return
	}
}

func (a *agentT) dispatch(channel any, event string) {
	if a.dispatcher != nil {
		a.dispatcher.Dispatch(a, channel, event)
	}
}

func (a *agentT) reviseTicker(resolver *content.Resolution, s messaging.Spanner) {

}

func (a *agentT) emissaryShutdown() {
	a.emissary.Close()
	a.ticker.Stop()
}

func (a *agentT) masterShutdown() {
	a.master.Close()
}

func (a *agentT) configure(m *messaging.Message) {
	switch m.ContentType() {
	case messaging.ContentTypeDispatcher:
		if dispatcher, ok := messaging.DispatcherContent(m); ok {
			a.dispatcher = dispatcher
		}
	}
	messaging.Reply(m, messaging.StatusOK(), a.Uri())
}
