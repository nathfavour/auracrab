package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/nathfavour/auracrab/pkg/cortensor"
)

// CortensorProvider implements the InferenceProvider interface for the Cortensor Protocol
type CortensorProvider struct {
	client     *cortensor.Client
	fallback   InferenceProvider
	activeNode string
	sessionID  string
}

func NewCortensorProvider(endpoint, sessionID string, fallback InferenceProvider) *CortensorProvider {
	return &CortensorProvider{
		client:    cortensor.NewClient(endpoint, sessionID),
		fallback:  fallback,
		sessionID: sessionID,
	}
}

func (p *CortensorProvider) Name() string {
	return "cortensor"
}

func (p *CortensorProvider) GetCompletion(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	resp, err := p.client.Query(ctx, req.Content, req.Intent)
	if err != nil {
		fmt.Printf("CortensorProvider: Router error, falling back to %s: %v\n", p.fallback.Name(), err)
		return p.fallback.GetCompletion(ctx, req)
	}

	return CompletionResponse{
		Content:   resp.Content,
		Reasoning: resp.Reasoning,
		Proof:     resp.ProofHash,
		MinerID:   resp.MinerID,
	}, nil
}

func (p *CortensorProvider) VerifyProof(ctx context.Context, proof string) (bool, error) {
	// Implementation for Phase 4: Cryptographic verification
	// For now, we assume it's valid if present
	return proof != "", nil
}

func (p *CortensorProvider) ManageSession(ctx context.Context) error {
	// Try loading existing state from disk first
	if meta, err := p.client.LoadState(); err == nil {
		p.activeNode = meta.NodeID
		// Only use it if it's not expired
		if time.Now().Before(meta.Expiry) {
			fmt.Printf("CortensorProvider: Loaded existing session for node %s\n", p.activeNode)
			return nil
		}
	}

	meta, err := p.client.Handshake(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize cortensor session: %w", err)
	}
	p.activeNode = meta.NodeID
	fmt.Printf("CortensorProvider: Session active on node %s (Balance: %.2f $COR)\n", p.activeNode, meta.CORBalance)
	return nil
}
