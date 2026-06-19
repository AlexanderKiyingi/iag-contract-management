// Package realtime provides the workspace WebSocket: a live push of the
// per-session workspace snapshot whenever any client mutates platform state.
//
// The browser opens wss://<gateway>/api/v1/contract-management/v1/ws/workspace
// with the access token as a ?token= query param (browsers cannot set an
// Authorization header on a WebSocket). The gateway forwards the upgrade and
// the platform-auth middleware authenticates the token, so by the time ServeWS
// runs the request context already carries the caller's models.Session.
//
// On connect the hub sends the caller's filtered snapshot; thereafter every
// successful mutating request triggers a Broadcast that re-projects the
// workspace per connection (each client only ever sees what its own session is
// permitted to see). When the socket is offline the UI falls back to /v1/bootstrap
// polling, so this is a progressive enhancement, never a hard dependency.
package realtime

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
)

const (
	// writeWait bounds a single write; pongWait is how long we'll wait for a
	// pong before considering the peer dead; pingPeriod must be < pongWait so a
	// pong always lands before the read deadline elapses. Mirrors the values in
	// shared/services/notifications so heartbeat timing is uniform across hubs.
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

// Push is the message envelope sent to clients. Mirrors the platform shape used
// by iag-project-management's workspace socket so the frontend handling is
// uniform: {type, data, version}.
type Push struct {
	Type    string          `json:"type"`
	Data    json.RawMessage `json:"data"`
	Version int64           `json:"version"`
}

var upgrader = websocket.Upgrader{
	// The gateway terminates origin/TLS and is the only public entry; cross
	// origin is enforced upstream, so accept the already-authenticated upgrade.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// clientState carries the connection's session plus a write mutex. gorilla
// permits one concurrent reader and one concurrent writer per conn; the mutex
// serializes the read-loop's keepalive writes against concurrent broadcasts.
type clientState struct {
	sess    models.Session
	writeMu sync.Mutex
}

func (cs *clientState) write(conn *websocket.Conn, payload []byte) error {
	cs.writeMu.Lock()
	defer cs.writeMu.Unlock()
	_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
	return conn.WriteMessage(websocket.TextMessage, payload)
}

// ping sends a websocket control ping under the same write mutex as snapshot
// writes — gorilla permits only one concurrent writer per connection.
func (cs *clientState) ping(conn *websocket.Conn) error {
	cs.writeMu.Lock()
	defer cs.writeMu.Unlock()
	_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
	return conn.WriteMessage(websocket.PingMessage, nil)
}

// Hub tracks live workspace connections and fans out snapshots.
type Hub struct {
	store   *models.Store
	mu      sync.RWMutex
	clients map[*websocket.Conn]*clientState
	version int64
}

// NewHub builds a hub backed by the workspace store.
func NewHub(store *models.Store) *Hub {
	return &Hub{store: store, clients: map[*websocket.Conn]*clientState{}}
}

// ServeWS upgrades the request and streams workspace snapshots until the client
// disconnects. Auth is already done by the platform-auth middleware, which puts
// the caller's session on the request context.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	if h == nil {
		http.Error(w, "realtime unavailable", http.StatusServiceUnavailable)
		return
	}
	sess, ok := models.RequestSession(r.Context())
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return // Upgrade already wrote the error response.
	}
	defer conn.Close()

	cs := &clientState{sess: sess}
	h.mu.Lock()
	h.clients[conn] = cs
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.clients, conn)
		h.mu.Unlock()
	}()

	// Initial snapshot so the client renders immediately without a bootstrap GET.
	_ = cs.write(conn, h.encode(sess))

	// Heartbeat: ping on an interval and require a pong inside pongWait, so a
	// silently dropped connection (idle past an LB/proxy timeout) is detected
	// instead of lingering as a dead entry that fails on the next broadcast.
	conn.SetReadLimit(512)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	done := make(chan struct{})
	defer close(done)
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if err := cs.ping(conn); err != nil {
					_ = conn.Close() // unblocks the read loop below
					return
				}
			}
		}
	}()

	// Drain inbound frames (the UI only sends pings/pongs); any read error —
	// including a missed pong tripping the read deadline — means disconnect.
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

// Broadcast re-projects the workspace for every connected session and pushes it.
// Call after any successful mutation. Safe to call concurrently.
func (h *Hub) Broadcast() {
	if h == nil {
		return
	}
	v := atomic.AddInt64(&h.version, 1)
	h.mu.RLock()
	targets := make([]*websocket.Conn, 0, len(h.clients))
	states := make([]*clientState, 0, len(h.clients))
	for conn, cs := range h.clients {
		targets = append(targets, conn)
		states = append(states, cs)
	}
	h.mu.RUnlock()
	for i, conn := range targets {
		cs := states[i]
		if err := cs.write(conn, h.encodeVersion(cs.sess, v)); err != nil {
			slog.Debug("workspace ws write", "err", err)
		}
	}
}

func (h *Hub) encode(sess models.Session) []byte {
	return h.encodeVersion(sess, atomic.LoadInt64(&h.version))
}

func (h *Hub) encodeVersion(sess models.Session, version int64) []byte {
	ws := h.store.GetWorkspaceForSession(sess)
	data, _ := json.Marshal(ws)
	payload, _ := json.Marshal(Push{Type: "workspace", Data: data, Version: version})
	return payload
}
