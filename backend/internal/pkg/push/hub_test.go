package push

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"couple-mini/backend/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func TestNotifyPairConfirmedSendsEventToSharingUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hub := NewHub()
	router := gin.New()
	router.GET("/events", func(c *gin.Context) {
		hub.Serve(c, "share-user")
	})
	server := httptest.NewServer(router)
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/events"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	waitForClient(t, hub, "share-user")
	hub.NotifyPairConfirmed(domain.Couple{
		ID:      "couple-1",
		UserAID: "share-user",
		UserBID: "receiver-user",
	})

	if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	var event Event
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatalf("read event: %v", err)
	}
	if event.Type != "pair:confirmed" {
		t.Fatalf("event.Type = %q, want pair:confirmed", event.Type)
	}
}

func TestNotifyPairUnboundSendsEventToPairedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hub := NewHub()
	router := gin.New()
	router.GET("/events", func(c *gin.Context) {
		hub.Serve(c, "receiver-user")
	})
	server := httptest.NewServer(router)
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/events"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	waitForClient(t, hub, "receiver-user")
	hub.NotifyPairUnbound(domain.UnpairResult{
		Couple: domain.Couple{
			ID:      "couple-1",
			UserAID: "share-user",
			UserBID: "receiver-user",
		},
		InitiatorID: "share-user",
	})

	if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	var event Event
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatalf("read event: %v", err)
	}
	if event.Type != "pair:unbound" {
		t.Fatalf("event.Type = %q, want pair:unbound", event.Type)
	}
}

func TestNotifyNoticeSendsEventToRecipient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hub := NewHub()
	router := gin.New()
	router.GET("/events", func(c *gin.Context) {
		hub.Serve(c, "receiver-user")
	})
	server := httptest.NewServer(router)
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/events"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	waitForClient(t, hub, "receiver-user")
	hub.NotifyNotice(domain.Notice{
		ID:          "notice-1",
		RecipientID: "receiver-user",
		InitiatorID: "sender-user",
		Category:    "todos",
		Title:       "新的待办任务",
		Content:     "买牛奶",
		CreatedAt:   time.Now(),
	})

	if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	var event Event
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatalf("read event: %v", err)
	}
	if event.Type != "app:notice" {
		t.Fatalf("event.Type = %q, want app:notice", event.Type)
	}
}

func waitForClient(t *testing.T, hub *Hub, userID string) {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		hub.mu.RLock()
		count := len(hub.clients[userID])
		hub.mu.RUnlock()
		if count > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("websocket client %q did not register", userID)
}
