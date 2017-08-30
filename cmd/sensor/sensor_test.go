package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/capsule8/reactive8/pkg/pubsub/mock"

	api "github.com/capsule8/api/v0"
	"github.com/capsule8/reactive8/pkg/config"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
)

func TestMain(m *testing.M) {
	flag.Parse()
	config.Sensor.Pubsub = "mock"

	os.Exit(m.Run())
}

// TestCreateSubscription tests for the successful creation of a subscription over NATS.
// It verifies sub creation by ensuring the delivery of a single message over the sub STAN channel.
func TestCreateSubscription(t *testing.T) {
	sub := &api.Subscription{
		EventFilter: &api.EventFilter{
			ChargenEvents: []*api.ChargenEventFilter{
				&api.ChargenEventFilter{
					Length: 1,
				},
			},
		},

		Modifier: &api.Modifier{
			Throttle: &api.ThrottleModifier{
				Interval:     1,
				IntervalType: 0,
			},
		},
	}
	mock.SetMockReturn("subscription.*", sub)

	b, _ := proto.Marshal(sub)
	h := sha256.New()
	h.Write(b)
	subID := fmt.Sprintf("%x", h.Sum(nil))

	s, err := CreateSensor()
	if err != nil {
		glog.Fatal("Error creating sensor:", err)
	}
	err = s.Start()
	if err != nil {
		glog.Fatal("Error starting sensor:", err)
	}
	stopSignal := make(chan interface{})

	msgs := make(chan *mock.OutboundMessage)
	go func() {
		timer := time.NewTimer(0)
	getMessageLoop:
		for {
			select {
			case <-stopSignal:
				break getMessageLoop
			case <-timer.C:
				if len(mock.GetOutboundMessages(fmt.Sprintf("event.%s", subID))) > 0 {
					// We only care about getting a single event here
					msgs <- &mock.GetOutboundMessages(fmt.Sprintf("event.%s", subID))[0]
				}
				timer.Reset(10 * time.Millisecond)
			}
		}

	}()

	select {
	case <-time.After(3 * time.Second):
		t.Error("Receive msg timeout")
	case ev := <-msgs:
		t.Log(ev.Topic)
		t.Log("Received message:", ev)
	}

	close(stopSignal)
	s.Shutdown()
	// Clear mock values after we're done
	mock.ClearMockValues()
	s.Wait()
}

// TestDiscover tests the discovery broadcast functionality in the sensor
func TestDiscover(t *testing.T) {
	s, err := CreateSensor()
	if err != nil {
		glog.Fatal("Error creating sensor:", err)
	}
	err = s.Start()
	if err != nil {
		glog.Fatal("Error starting sensor:", err)
	}

	msgs := make(chan *mock.OutboundMessage)
	stopSignal := make(chan interface{})
	go func() {
		timer := time.NewTimer(0)
		defer timer.Stop()
	getMessageLoop:
		for {
			select {
			case <-stopSignal:
				break getMessageLoop
			case <-timer.C:
				if len(mock.GetOutboundMessages("discover.sensor")) > 0 {
					// We only care about getting a single event here
					msgs <- &mock.GetOutboundMessages("discover.sensor")[0]
				}
				timer.Reset(10 * time.Millisecond)
			}
		}

	}()

	select {
	case <-time.After(3 * time.Second):
		t.Error("Receive msg timeout")
	case ev := <-msgs:
		t.Log("Received message:", ev)
	}

	close(stopSignal)
	s.Shutdown()
	// Clear mock values after we're done
	mock.ClearMockValues()
	s.Wait()
}
