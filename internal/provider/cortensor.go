package provider

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nathfavour/auracrab/pkg/cortensor"
)

// CortensorProvider implements the InferenceProvider interface for the Cortensor Protocol
type CortensorProvider struct {
	client             *cortensor.Client
	fallback           InferenceProvider
	activeNode         string
	sessionID          string
	consensusThreshold int
	mu                 sync.RWMutex
}

func NewCortensorProvider(endpoint, sessionID string, threshold int, fallback InferenceProvider) *CortensorProvider {
	return &CortensorProvider{
		client:             cortensor.NewClient(endpoint, sessionID),
		fallback:           fallback,
		sessionID:          sessionID,
		consensusThreshold: threshold,
	}
}

func (p *CortensorProvider) Name() string {
	return "cortensor"
}

func (p *CortensorProvider) GetCompletion(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	threshold := p.consensusThreshold
	if threshold <= 1 {
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

	// N-of-M Consensus Logic
	type result struct {
		resp *cortensor.RouterCompletionResponse
		err  error
	}
	resChan := make(chan result, threshold)

	for i := 0; i < threshold; i++ {
		go func() {
			r, err := p.client.Query(ctx, req.Content, req.Intent)
			resChan <- result{r, err}
		}()
	}

	responses := []*cortensor.RouterCompletionResponse{}
	for i := 0; i < threshold; i++ {
		res := <-resChan
		if res.err == nil {
			responses = append(responses, res.resp)
		}
	}

	if len(responses) == 0 {
		return p.fallback.GetCompletion(ctx, req)
	}

	return p.majorityVote(responses), nil
}

func (p *CortensorProvider) majorityVote(resps []*cortensor.RouterCompletionResponse) CompletionResponse {
	// Simple consensus: most frequent content
	counts := make(map[string]int)
	bestIdx := 0
	maxCount := 0

	for i, r := range resps {
		counts[r.Content]++
		if counts[r.Content] > maxCount {
			maxCount = counts[r.Content]
			bestIdx = i
		}
	}

	best := resps[bestIdx]
	return CompletionResponse{
		Content:   best.Content,
		Reasoning: best.Reasoning,
		Proof:     best.ProofHash,
		MinerID:   fmt.Sprintf("%s (+ %d others)", best.MinerID, len(resps)-1),
	}
}

func (p *CortensorProvider) VerifyProof(ctx context.Context, proof string) (bool, error) {
	// Implementation for Phase 4: Cryptographic verification
	// For now, we assume it's valid if present
	return proof != "", nil
}

func (p *CortensorProvider) ManageSession(ctx context.Context) error {
	// Try loading existing state from disk first
	if meta, err := p.client.LoadState(); err == nil {
		p.mu.Lock()
		p.activeNode = meta.NodeID
		p.mu.Unlock()
		// Only use it if it's not expired
		if time.Now().Before(meta.Expiry) {
			fmt.Printf("CortensorProvider: Loaded existing session for node %s\n", meta.NodeID)
			return nil
		}
	}

	meta, err := p.client.Handshake(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize cortensor session: %w", err)
	}
	p.mu.Lock()
	p.activeNode = meta.NodeID
	p.mu.Unlock()
	fmt.Printf("CortensorProvider: Session active on node %s (Balance: %.2f $COR)\n", meta.NodeID, meta.CORBalance)
	return nil
}

func (p *CortensorProvider) GetInfo() string {
	p.mu.RLock()
	node := p.activeNode
	p.mu.RUnlock()

	if node == "" {
		return "Connecting..."
	}
	meta, err := p.client.LoadState()
	if err != nil {
		return node
	}
	return fmt.Sprintf("%s (%.2f $COR)", node, meta.CORBalance)
}
