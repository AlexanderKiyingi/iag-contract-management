// Package events publishes contract-management domain events to iag.commercial.
// Alert-shaped events (contracts.alert.raised) are consumed by iag-notifications.
package events

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

const (
	SpecVersion = "1.0"
	Source      = "iag.contract-management"
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
	body, err := json.Marshal(evt)
	if err != nil {
		slog.Warn("contract event marshal failed", "type", eventType, "err", err)
		return
	}
	if key == "" {
		key = evt.ID
	}
	if err := b.writer.WriteMessages(ctx, kafka.Message{
		Topic: TopicCommercial,
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

// PublishAlert emits contracts.alert.raised for iag-notifications policy consumers.
func (b *Bus) PublishAlert(ctx context.Context, channel, recipient, templateID string, variables map[string]string, key string) {
	if !b.enabled || recipient == "" || templateID == "" {
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
