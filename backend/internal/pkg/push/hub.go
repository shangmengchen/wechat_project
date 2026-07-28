package push

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"couple-mini/backend/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

type Event struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

type Hub struct {
	upgrader websocket.Upgrader
	mu       sync.RWMutex
	clients  map[string]map[*client]struct{}
}

type client struct {
	hub    *Hub
	userID string
	conn   *websocket.Conn
	send   chan Event
}

func NewHub() *Hub {
	return &Hub{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		clients: make(map[string]map[*client]struct{}),
	}
}

func (h *Hub) Serve(c *gin.Context, userID string) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"code":    http.StatusUnauthorized,
			"message": "unauthorized",
		})
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	cli := &client{
		hub:    h,
		userID: userID,
		conn:   conn,
		send:   make(chan Event, 8),
	}
	h.register(cli)

	go cli.writePump()
	cli.readPump()
}

func (h *Hub) NotifyPairConfirmed(couple domain.Couple) {
	event := Event{
		Type: "pair:confirmed",
		Data: gin.H{
			"coupleId": couple.ID,
			"userAId":  couple.UserAID,
			"userBId":  couple.UserBID,
		},
	}

	h.notifyCoupleUsers(couple, event)
}

func (h *Hub) NotifyPairUnbound(result domain.UnpairResult) {
	event := Event{
		Type: "pair:unbound",
		Data: gin.H{
			"coupleId":    result.Couple.ID,
			"userAId":     result.Couple.UserAID,
			"userBId":     result.Couple.UserBID,
			"initiatorId": result.InitiatorID,
		},
	}
	h.notifyCoupleUsers(result.Couple, event)
}

func (h *Hub) NotifyNotice(notice domain.Notice) {
	if strings.TrimSpace(notice.RecipientID) == "" {
		return
	}
	h.notifyUser(notice.RecipientID, Event{
		Type: "app:notice",
		Data: notice,
	})
}

func (h *Hub) notifyCoupleUsers(couple domain.Couple, event Event) {
	notified := map[string]struct{}{}
	for _, userID := range []string{couple.UserAID, couple.UserBID} {
		userID = strings.TrimSpace(userID)
		if userID == "" {
			continue
		}
		if _, ok := notified[userID]; ok {
			continue
		}
		notified[userID] = struct{}{}
		h.notifyUser(userID, event)
	}
}

func (h *Hub) register(cli *client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[cli.userID] == nil {
		h.clients[cli.userID] = make(map[*client]struct{})
	}
	h.clients[cli.userID][cli] = struct{}{}
}

func (h *Hub) unregister(cli *client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	clients := h.clients[cli.userID]
	if clients == nil {
		return
	}
	if _, ok := clients[cli]; !ok {
		return
	}
	delete(clients, cli)
	close(cli.send)
	if len(clients) == 0 {
		delete(h.clients, cli.userID)
	}
}

func (h *Hub) notifyUser(userID string, event Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for cli := range h.clients[userID] {
		select {
		case cli.send <- event:
		default:
		}
	}
}

func (cli *client) readPump() {
	defer func() {
		cli.hub.unregister(cli)
		_ = cli.conn.Close()
	}()

	cli.conn.SetReadLimit(512)
	_ = cli.conn.SetReadDeadline(time.Now().Add(pongWait))
	cli.conn.SetPongHandler(func(string) error {
		_ = cli.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		if _, _, err := cli.conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (cli *client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = cli.conn.Close()
	}()

	for {
		select {
		case event, ok := <-cli.send:
			_ = cli.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = cli.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := cli.conn.WriteJSON(event); err != nil {
				return
			}
		case <-ticker.C:
			_ = cli.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := cli.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
