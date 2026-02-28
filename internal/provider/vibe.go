package provider

import (
	"context"
	"fmt"

	"github.com/nathfavour/auracrab/pkg/vibe"
)

// VibeProvider is the default/fallback provider that uses the local vibeauracle UDS.
type VibeProvider struct {
	client *vibe.Client
}

func NewVibeProvider() *VibeProvider {
	return &VibeProvider{
		client: vibe.NewClient(),
	}
}

func (p *VibeProvider) Name() string {
	return "vibe"
}

func (p *VibeProvider) GetCompletion(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	content, err := p.client.Query(req.Content, req.Intent)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("vibe provider error: %w", err)
	}

	return CompletionResponse{
		Content: content,
	}, nil
}

func (p *VibeProvider) VerifyProof(ctx context.Context, proof string) (bool, error) {
	// Local Vibe provider doesn't use cryptographic proofs.
	return true, nil
}

func (p *VibeProvider) ManageSession(ctx context.Context) error {
	// Vibe provider is a simple stateless UDS connection.
	return nil
}

func (p *VibeProvider) GetInfo() string {
	return "Local (UDS)"
}
