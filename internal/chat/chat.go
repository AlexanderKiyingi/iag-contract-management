// Package chat is a thin contract-management wrapper over the iag-chat-client
// S2S library. It find-or-creates a discussion thread per governance contract
// and posts system lines. When chat is not configured (no CHAT_API_URL or
// service secret) New returns nil and all methods no-op, so the caller never
// has to branch.
package chat

import (
	"context"

	chatclient "github.com/alvor-technologies/iag-chat-client"

	"github.com/alvor-technologies/iag-contract-management/internal/config"
)

const (
	linkService    = "contract-management"
	linkEntityType = "contract"
)

// Service wraps the chat client for contract threads.
type Service struct {
	c *chatclient.Client
}

// New returns a Service, or nil when chat is not configured. A nil Service is
// safe to call — every method no-ops.
func New(cfg config.Config) *Service {
	if cfg.ChatAPIURL == "" || cfg.ServiceClientSecret == "" {
		return nil
	}
	return &Service{
		c: chatclient.New(chatclient.Options{
			BaseURL:      cfg.ChatAPIURL,
			TokenURL:     cfg.AuthTokenURL,
			ClientID:     cfg.ServiceClientID,
			ClientSecret: cfg.ServiceClientSecret,
		}),
	}
}

func contractLink(contractID string) chatclient.Link {
	return chatclient.Link{Service: linkService, EntityType: linkEntityType, EntityID: contractID}
}

// EnsureContractThread find-or-creates a contract's discussion thread and
// returns its conversation id. Safe on a nil Service (returns "", nil).
func (s *Service) EnsureContractThread(ctx context.Context, contractID, title string, participants []string) (string, error) {
	if s == nil {
		return "", nil
	}
	conv, err := s.c.UpsertThread(ctx, contractLink(contractID), title, participants...)
	if err != nil {
		return "", err
	}
	return conv.ID, nil
}

// PostSystem posts a system message into a contract's thread. Safe on nil.
func (s *Service) PostSystem(ctx context.Context, contractID, message string) error {
	if s == nil {
		return nil
	}
	return s.c.PostSystem(ctx, contractLink(contractID), message)
}
