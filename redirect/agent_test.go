package redirect

import (
	"fmt"
	"github.com/behavioral-ai/collective/operations/operationstest"
	"github.com/behavioral-ai/core/messaging"
	"github.com/behavioral-ai/traffic/redirect/representation1"
	"github.com/behavioral-ai/traffic/timeseries"
	"time"
)

func ExampleNewAgent() {
	a := newAgent(representation1.Initialize(nil), operationstest.NewService())

	fmt.Printf("test: newAgent() -> [%v}\n", a.Name())

	//Output:
	//test: newAgent() -> [test:resiliency:agent/redirect/request/http}

}

func _ExampleAgent_LoadContent() {
	ch := make(chan struct{})
	agent := newAgent(representation1.Initialize(nil), operationstest.NewService())
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
	agent := newAgent(representation1.Initialize(nil), operationstest.NewService())
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
	agent := newAgent(representation1.Initialize(nil), operationstest.NewService())
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
