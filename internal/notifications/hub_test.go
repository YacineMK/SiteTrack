package notifications

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHubPublishToSubscribers(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()

	first, unsubscribeFirst := hub.Subscribe(userID)
	defer unsubscribeFirst()
	second, unsubscribeSecond := hub.Subscribe(userID)
	defer unsubscribeSecond()

	notification := NotificationResponse{Title: "hello"}
	hub.Publish(userID, notification)

	for name, events := range map[string]<-chan NotificationResponse{"first": first, "second": second} {
		select {
		case got := <-events:
			if got.Title != "hello" {
				t.Fatalf("%s received %q, want hello", name, got.Title)
			}
		case <-time.After(time.Second):
			t.Fatalf("%s did not receive notification", name)
		}
	}
}

func TestHubUnsubscribeClosesChannel(t *testing.T) {
	hub := NewHub()
	userID := uuid.New()

	events, unsubscribe := hub.Subscribe(userID)
	unsubscribe()

	if _, ok := <-events; ok {
		t.Fatal("channel still open after unsubscribe")
	}

	hub.Publish(userID, NotificationResponse{Title: "ignored"})
}

func TestHubIsolatesUsers(t *testing.T) {
	hub := NewHub()
	subscriberID := uuid.New()
	otherID := uuid.New()

	events, unsubscribe := hub.Subscribe(subscriberID)
	defer unsubscribe()

	hub.Publish(otherID, NotificationResponse{Title: "other"})
	hub.Publish(subscriberID, NotificationResponse{Title: "mine"})

	select {
	case got := <-events:
		if got.Title != "mine" {
			t.Fatalf("received %q, want mine", got.Title)
		}
	case <-time.After(time.Second):
		t.Fatal("did not receive notification")
	}
}
