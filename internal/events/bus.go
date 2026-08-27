// Package events publishes contract-management domain events to iag.commercial.
// Alert-shaped events (contracts.alert.raised) are consumed by iag-notifications.
package events

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/alvor-technologies/iag-contract-management/internal/outbox"
)

const (
	SpecVersion     = "1.0"
	Source          = "iag.contract-management"
	TopicCommercial = "iag.commercial"

	TypeContractCreated       = "contracts.contract.created"
	TypeContractUpdated       = "contracts.contract.updated"
	TypeContractDeleted       = "contracts.contract.deleted"
	TypeContractStatusChanged = "contracts.contract.status_changed"
	TypeAssistanceRequested   = "contracts.assistance.requested"
	TypeMilestoneDueSoon      = "contracts.milestone.due_soon"
	TypeAlertRaised           = "contracts.alert.raised"
)

type Bus struct {
	writer  *kafka.Writer
	enabled bool
	// outbox, when set, makes PublishCommercial durable: events are persisted
	// and drained to Kafka by a background publisher instead of being written
	// inline. nil falls back to the legacy direct-write path (used by tests).
	outbox *outbox.Store
}

// UseOutbox switches the Bus onto the durable outbox path. Call once at boot
// after the Postgres pool is ready; the caller is responsible for starting an
// outbox.Publisher that drains the table.
func (b *Bus) UseOutbox(s *outbox.Store) {
	if b != nil {
		b.outbox = s
	}
}

type Config struct {
	Brokers []string
	Enabled bool
}

func NewFromEnv() *Bus {
	return New(Config{
		Brokers: ParseBrokers(os.Getenv("KAFKA_BROKERS")),
		Enabled: strings.EqualFold(os.Getenv("EVENT_BUS_ENABLED"), "true"),
	})
}

func New(cfg Config) *Bus {
	if !cfg.Enabled || len(cfg.Brokers) == 0 {
		return &Bus{enabled: false}
	}
	return &Bus{
		enabled: true,
		writer: &kafka.Writer{
			Addr:         kafka.TCP(cfg.Brokers...),
			Topic:        TopicCommercial,
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireAll,
			Transport:    &kafka.Transport{ClientID: Source},
		},
	}
}

func (b *Bus) Close() error {
	if b == nil || !b.enabled || b.writer == nil {
		return nil
	}
	return b.writer.Close()
}

func (b *Bus) Enabled() bool { return b != nil && b.enabled }

type PlatformEvent struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"`
	Time        string         `json:"time"`
	Source      string         `json:"source"`
	SpecVersion string         `json:"specversion"`
	Data        map[string]any `json:"data"`
}

// PublishCommercialTx enqueues the event and reports whether it landed.
//
// PublishCommercial swallows the enqueue error, which is right for an event
// that merely notifies. It is wrong for one that carries a financial
// consequence: if the row cannot be written, the caller must not commit the
// change that the event was supposed to announce. Used with GovStore.InTx, a
// failure here rolls the whole thing back and the client retries.
func (b *Bus) PublishCommercialTx(ctx context.Context, eventType string, data map[string]any, key string) error {
	if !b.enabled || b.outbox == nil {
		// No durable path configured (unit tests, or events disabled). Fall back
		// to the best-effort publish rather than failing the business write.
		b.PublishCommercial(ctx, eventType, data, key)
		return nil
	}
	evt := PlatformEvent{
		ID:          uuid.NewString(),
		Type:        eventType,
		Time:        time.Now().UTC().Format(time.RFC3339Nano),
		Source:      Source,
		SpecVersion: SpecVersion,
		Data:        data,
	}
	if key == "" {
		key = evt.ID
	}
	return b.outbox.Enqueue(ctx, eventType, key, evt)
}

func (b *Bus) PublishCommercial(ctx context.Context, eventType string, data map[string]any, key string) {
	if !b.enabled || b.writer == nil {
		return
	}
	evt := PlatformEvent{
		ID:          uuid.NewString(),
		Type:        eventType,
		Time:        time.Now().UTC().Format(time.RFC3339Nano),
		Source:      Source,
		SpecVersion: SpecVersion,
		Data:        data,
	}
	if key == "" {
		key = evt.ID
	}

	// Durable path: persist the event; the background publisher drains it to
	// Kafka with retry, so a broker outage delays delivery instead of losing
	// it. DispatchOutbox below performs the actual write.
	if b.outbox != nil {
		if err := b.outbox.Enqueue(ctx, eventType, key, evt); err != nil {
			slog.Warn("contract event enqueue failed", "type", eventType, "err", err)
		}
		return
	}

	// Legacy direct path (outbox not configured, e.g. unit tests).
	body, err := json.Marshal(evt)
	if err != nil {
		slog.Warn("contract event marshal failed", "type", eventType, "err", err)
		return
	}
	if err := // No Topic on the Message: the writer is dedicated to one topic and already
		// carries it, and kafka-go rejects a Message that sets Topic as well ("Topic
		// must not be specified for both Writer and Message"). It fails before
		// sending, so every publish errored while looking transient to the retry loop.
		b.writer.WriteMessages(ctx, kafka.Message{
			Key:   []byte(key),
			Value: body,
			Headers: []kafka.Header{
				{Key: "ce-type", Value: []byte(eventType)},
				{Key: "ce-source", Value: []byte(Source)},
			},
		}); err != nil {
		slog.Warn("contract event publish failed", "type", eventType, "err", err)
	}
}

// DispatchOutbox writes a persisted outbox row to Kafka. It is the
// outbox.Dispatcher implementation the background publisher calls. The row
// payload is the already-marshaled PlatformEvent, so this just frames it with
// the routing key and CloudEvents headers.
func (b *Bus) DispatchOutbox(ctx context.Context, row outbox.Row) error {
	if !b.enabled || b.writer == nil {
		// Bus disabled: treat as delivered so the row isn't retried forever.
		return nil
	}
	// No Topic on the Message: the writer is dedicated to one topic and already
	// carries it, and kafka-go rejects a Message that sets Topic as well ("Topic
	// must not be specified for both Writer and Message"). It fails before
	// sending, so every publish errored while looking transient to the retry loop.
	return b.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(row.EventKey),
		Value: row.Payload,
		Headers: []kafka.Header{
			{Key: "ce-type", Value: []byte(row.EventType)},
			{Key: "ce-source", Value: []byte(Source)},
		},
	})
}

// PublishAlert emits contracts.alert.raised for iag-notifications policy consumers.
func (b *Bus) PublishAlert(ctx context.Context, channel, recipient, templateID string, variables map[string]string, key string) {
	b.PublishAlertTo(ctx, channel, "", recipient, templateID, variables, key)
}

// PublishAlertTo addresses a logical audience ("approvals.contracts") whose
// recipients an administrator maintains centrally, falling back to recipient
// until that audience is routed. Prefer it wherever the destination is a desk
// rather than a specific person.
func (b *Bus) PublishAlertTo(ctx context.Context, channel, audience, recipient, templateID string, variables map[string]string, key string) {
	if !b.enabled || templateID == "" {
		return
	}
	if recipient == "" && audience == "" {
		warnNoNotifyRecipient()
		return
	}
	vars := map[string]any{}
	for k, v := range variables {
		vars[k] = v
	}
	if channel == "" {
		channel = defaultNotifyChannel()
	}
	data := map[string]any{
		"channel":    channel,
		"recipient":  recipient,
		"templateId": templateID,
		"variables":  vars,
	}
	if audience != "" {
		data["audience"] = audience
	}
	b.PublishCommercial(ctx, TypeAlertRaised, data, key)
}

func ParseBrokers(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func defaultNotifyChannel() string {
	if ch := strings.TrimSpace(os.Getenv("NOTIFY_CHANNEL")); ch != "" {
		return ch
	}
	return "email"
}

func DefaultNotifyRecipient() string {
	return strings.TrimSpace(os.Getenv("NOTIFY_DEFAULT_RECIPIENT"))
}

func AssistanceTemplateID() string {
	if t := strings.TrimSpace(os.Getenv("TEMPLATE_CONTRACTS_ASSISTANCE")); t != "" {
		return t
	}
	return "contracts-assistance-requested"
}

func MilestoneDueTemplateID() string {
	if t := strings.TrimSpace(os.Getenv("TEMPLATE_CONTRACTS_MILESTONE_DUE")); t != "" {
		return t
	}
	return "contracts-milestone-due-soon"
}

var notifyRecipientWarnOnce sync.Once

// warnNoNotifyRecipient logs once when an alert is dropped for want of a
// recipient. Without it an unset NOTIFY_DEFAULT_RECIPIENT is indistinguishable
// from "no alerts were raised": the emitter returns early, nothing reaches the
// notifications service, and no error appears anywhere.
func warnNoNotifyRecipient() {
	notifyRecipientWarnOnce.Do(func() {
		slog.Warn("contracts alert dropped: no recipient and NOTIFY_DEFAULT_RECIPIENT is unset; contracts.alert.raised events will not be emitted")
	})
}
