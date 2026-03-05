package eventbus

import (
	"testing"
	"time"

	"github.com/anthropics/story/internal/shared/types"
)

// testGameEvent implements types.GameEvent for testing.
type testGameEvent struct {
	base types.BaseEvent
}

func (e testGameEvent) EventType() string            { return "test" }
func (e testGameEvent) GetBaseEvent() types.BaseEvent { return e.base }

func TestSubscribeAndPublishGameEvent(t *testing.T) {
	eb := NewEventBus()
	defer eb.Close()

	ch := eb.SubscribeGameEvent()
	evt := testGameEvent{base: types.BaseEvent{ID: "evt-1"}}

	eb.PublishGameEvent(evt)

	select {
	case received := <-ch:
		if received.GetBaseEvent().ID != "evt-1" {
			t.Fatalf("expected event ID evt-1, got %s", received.GetBaseEvent().ID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for game event")
	}
}

func TestSubscribeAndPublishChat(t *testing.T) {
	eb := NewEventBus()
	defer eb.Close()

	ch := eb.SubscribeChat()
	data := ChatData{SenderID: "p1", Content: "hello", Scope: "global"}

	eb.PublishChat(data)

	select {
	case received := <-ch:
		if received.SenderID != "p1" || received.Content != "hello" {
			t.Fatalf("unexpected chat data: %+v", received)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for chat event")
	}
}

func TestSubscribeAndPublishPlayerConnected(t *testing.T) {
	eb := NewEventBus()
	defer eb.Close()

	ch := eb.SubscribePlayerConnected()
	eb.PublishPlayerConnected(PlayerConnectedData{PlayerID: "p1"})

	select {
	case received := <-ch:
		if received.PlayerID != "p1" {
			t.Fatalf("expected player ID p1, got %s", received.PlayerID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
}

func TestSubscribeAndPublishPlayerDisconnected(t *testing.T) {
	eb := NewEventBus()
	defer eb.Close()

	ch := eb.SubscribePlayerDisconnected()
	eb.PublishPlayerDisconnected(PlayerDisconnectedData{PlayerID: "p2"})

	select {
	case received := <-ch:
		if received.PlayerID != "p2" {
			t.Fatalf("expected player ID p2, got %s", received.PlayerID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
}

func TestSubscribeAndPublishStateChanged(t *testing.T) {
	eb := NewEventBus()
	defer eb.Close()

	ch := eb.SubscribeStateChanged()
	eb.PublishStateChanged(StateChangedData{ChangeType: "player_moved", Data: "room-1"})

	select {
	case received := <-ch:
		if received.ChangeType != "player_moved" {
			t.Fatalf("expected change type player_moved, got %s", received.ChangeType)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
}

func TestSubscribeAndPublishGameStatusChanged(t *testing.T) {
	eb := NewEventBus()
	defer eb.Close()

	ch := eb.SubscribeGameStatusChanged()
	eb.PublishGameStatusChanged(GameStatusChangedData{
		From: types.GameStatusLobby,
		To:   types.GameStatusPlaying,
	})

	select {
	case received := <-ch:
		if received.From != types.GameStatusLobby || received.To != types.GameStatusPlaying {
			t.Fatalf("unexpected status change: %+v", received)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
}

func TestSubscribeAndPublishSendEndings(t *testing.T) {
	eb := NewEventBus()
	defer eb.Close()

	ch := eb.SubscribeSendEndings()
	eb.PublishSendEndings(types.GameEndData{CommonResult: "everyone wins"})

	select {
	case received := <-ch:
		if received.CommonResult != "everyone wins" {
			t.Fatalf("unexpected ending: %+v", received)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
}

func TestSubscribeAndPublishFeedback(t *testing.T) {
	eb := NewEventBus()
	defer eb.Close()

	ch := eb.SubscribeFeedback()
	eb.PublishFeedback(types.Feedback{PlayerID: "p1", FunRating: 5})

	select {
	case received := <-ch:
		if received.PlayerID != "p1" || received.FunRating != 5 {
			t.Fatalf("unexpected feedback: %+v", received)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
}

func TestMultipleSubscribers(t *testing.T) {
	eb := NewEventBus()
	defer eb.Close()

	ch1 := eb.SubscribeChat()
	ch2 := eb.SubscribeChat()

	data := ChatData{SenderID: "p1", Content: "broadcast"}
	eb.PublishChat(data)

	for i, ch := range []<-chan ChatData{ch1, ch2} {
		select {
		case received := <-ch:
			if received.Content != "broadcast" {
				t.Fatalf("subscriber %d: unexpected content %s", i, received.Content)
			}
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d: timed out", i)
		}
	}
}

func TestOverflowDoesNotBlock(t *testing.T) {
	eb := NewEventBus()
	defer eb.Close()

	_ = eb.SubscribeChat()

	// Fill the buffer
	for i := 0; i < bufferSize; i++ {
		eb.PublishChat(ChatData{Content: "fill"})
	}

	// This should not block even though buffer is full
	done := make(chan struct{})
	go func() {
		eb.PublishChat(ChatData{Content: "overflow"})
		close(done)
	}()

	select {
	case <-done:
		// success - publish did not block
	case <-time.After(time.Second):
		t.Fatal("PublishChat blocked on full buffer")
	}
}

func TestCloseChannels(t *testing.T) {
	eb := NewEventBus()

	ch := eb.SubscribeChat()
	eb.Close()

	// Reading from closed channel should return zero value
	_, ok := <-ch
	if ok {
		t.Fatal("expected channel to be closed")
	}
}
