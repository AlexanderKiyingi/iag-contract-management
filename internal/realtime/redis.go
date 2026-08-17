package realtime

import (
	"context"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

// Cross-instance workspace nudges.
//
// Broadcast() re-projects the workspace from the store for each session this
// instance holds, and the middleware calls it after a successful mutation. That
// is complete on one instance and silently partial on two: a write handled by
// instance A refreshes only A's sockets, so anyone connected to B keeps looking
// at stale contract data until they happen to make a change themselves. There
// is no error and nothing in the logs — the screen is just wrong.
//
// The fix does not need to carry the change. Every instance reads the same
// database, so one instance only has to tell the others that something moved;
// each then re-projects for its own sessions. The message is deliberately
// empty — a nudge, not a payload — which means there is no schema to keep in
// step between versions during a rolling deploy.
const nudgeChannel = "cm:ws:workspace"

// NudgeBridge relays "the workspace changed" between instances.
type NudgeBridge struct {
	client *redis.Client
	hub    *Hub
}

func NewNudgeBridge(client *redis.Client, hub *Hub) *NudgeBridge {
	return &NudgeBridge{client: client, hub: hub}
}

// Notify tells every instance — including this one — that a mutation landed.
//
// The publisher does not broadcast locally as well: it receives its own message
// back through the subscription, so doing both would re-project every session
// on this instance twice for one write.
func (b *NudgeBridge) Notify(ctx context.Context) {
	if b == nil || b.client == nil {
		return
	}
	if err := b.client.Publish(ctx, nudgeChannel, "1").Err(); err != nil {
		// Redis is not a dependency of correctness for the instance that served
		// the write — fall back to refreshing our own sessions so that at worst
		// this behaves exactly as it did before the bridge existed.
		slog.Warn("workspace nudge publish failed; refreshing local sessions only", "err", err)
		b.hub.Broadcast()
	}
}

// RunSubscriber refreshes this instance's sessions whenever any instance
// reports a change. Returns when ctx is done.
func (b *NudgeBridge) RunSubscriber(ctx context.Context) {
	if b == nil || b.client == nil {
		return
	}
	pubsub := b.client.Subscribe(ctx, nudgeChannel)
	defer func() { _ = pubsub.Close() }()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-ch:
			if !ok {
				return
			}
			b.hub.Broadcast()
		}
	}
}
