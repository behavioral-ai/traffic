package redirect

import (
	"fmt"
	"github.com/behavioral-ai/collective/eventing/eventtest"
	"github.com/behavioral-ai/collective/timeseries"
	"github.com/behavioral-ai/core/messaging"
	"time"
)

func ExampleNewAgent() {
	a := newAgent(eventtest.New())

	fmt.Printf("test: newAgent() -> [uri:%v}\n", a.Uri())

	//Output:
	//test: newAgent() -> [uri:resiliency:agent/behavioral-ai/traffic/redirect}

}

func _ExampleAgent_LoadContent() {
	ch := make(chan struct{})
	agent := newAgent(eventtest.New())
	agent.dispatcher = messaging.NewTraceDispatcher()

	go func() {
		go masterAttend(agent, timeseries.Functions)
		go emissaryAttend(agent)
		time.Sleep(testDuration * 5)

		agent.Message(messaging.ShutdownMessage)
		time.Sleep(testDuration * 2)
		ch <- struct{}{}
	}()
	<-ch
	close(ch)

	//Output:
	//fail
}

func _ExampleAgent_NotFound() {
	ch := make(chan struct{})
	agent := newAgent(eventtest.New())
	agent.dispatcher = messaging.NewTraceDispatcher()

	go func() {
		agent.Message(messaging.StartupMessage)
		time.Sleep(testDuration * 5)
		agent.Message(messaging.ShutdownMessage)
		time.Sleep(testDuration * 2)
		ch <- struct{}{}
	}()
	<-ch
	close(ch)

	//Output:
	//fail
}

func _ExampleAgent_Resolver() {
	ch := make(chan struct{})
	agent := newAgent(eventtest.New())
	agent.dispatcher = messaging.NewTraceDispatcher()
	//test2.Startup()

	go func() {
		agent.Message(messaging.StartupMessage)
		time.Sleep(testDuration * 5)
		agent.Message(messaging.ShutdownMessage)
		time.Sleep(testDuration * 2)
		ch <- struct{}{}
	}()
	<-ch
	close(ch)

	//Output:
	//fail
}
