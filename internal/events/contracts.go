package events

import (
	"context"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
)

func PublishContractCreated(ctx context.Context, b *Bus, c models.Contract) {
	if b == nil || !b.Enabled() {
		return
	}
	b.PublishCommercial(ctx, TypeContractCreated, contractData(c), c.No)
}

func PublishContractUpdated(ctx context.Context, b *Bus, c models.Contract) {
	if b == nil || !b.Enabled() {
		return
	}
	b.PublishCommercial(ctx, TypeContractUpdated, contractData(c), c.No)
}

func PublishContractStatusChanged(ctx context.Context, b *Bus, c models.Contract, previous models.ContractStatus) {
	if b == nil || !b.Enabled() {
		return
	}
	data := contractData(c)
	data["previousStatus"] = string(previous)
	b.PublishCommercial(ctx, TypeContractStatusChanged, data, c.No)
}

func PublishContractDeleted(ctx context.Context, b *Bus, c models.Contract) {
	if b == nil || !b.Enabled() {
		return
	}
	b.PublishCommercial(ctx, TypeContractDeleted, contractData(c), c.No)
}

func PublishAssistanceRequested(ctx context.Context, b *Bus, msg models.AssistanceMessage) {
	if b == nil || !b.Enabled() {
		return
	}
	data := map[string]any{
		"from": msg.From,
		"text": msg.Text,
		"at":   msg.At,
	}
	b.PublishCommercial(ctx, TypeAssistanceRequested, data, msg.From)

	recipient := DefaultNotifyRecipient()
	if recipient != "" {
		b.PublishAlert(ctx, "", recipient, AssistanceTemplateID(), map[string]string{
			"sender": msg.From,
			"text":   msg.Text,
			"sentAt": msg.At,
		}, "assistance-"+msg.From+"-"+msg.At)
	}
}

func PublishMilestoneDueSoon(ctx context.Context, b *Bus, m models.Milestone, eventID string) {
	if b == nil || !b.Enabled() {
		return
	}
	data := map[string]any{
		"id":     m.ID,
		"title":  m.Title,
		"due":    m.Due,
		"zone":   m.Zone,
		"status": m.Status,
		"owner":  m.Owner,
	}
	b.PublishCommercial(ctx, TypeMilestoneDueSoon, data, m.ID)

	recipient := DefaultNotifyRecipient()
	if recipient != "" {
		if eventID == "" {
			eventID = "milestone-due-" + m.ID + "-" + m.Due
		}
		b.PublishAlert(ctx, "", recipient, MilestoneDueTemplateID(), map[string]string{
			"milestoneId": m.ID,
			"title":       m.Title,
			"due":         m.Due,
			"zone":        m.Zone,
			"owner":       m.Owner,
		}, eventID)
	}
}

func contractData(c models.Contract) map[string]any {
	return map[string]any{
		"no":      c.No,
		"name":    c.Name,
		"zone":    c.Zone,
		"status":  string(c.Status),
		"cs":      c.Cs,
		"paid":    c.Paid,
		"bal":     c.Bal,
		"prog":    c.Prog,
		"sup":     c.Sup,
		"created": c.Created,
	}
}
