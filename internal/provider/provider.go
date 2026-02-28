package provider

import (
	"context"
)

// CompletionRequest defines the input for an inference query
type CompletionRequest struct {
	Content string `json:"content"`
	Intent  string `json:"intent,omitempty"`
}

// CompletionResponse defines the output from an inference query
type CompletionResponse struct {
	Content   string `json:"content"`
	Reasoning string `json:"reasoning,omitempty"`
	Proof     string `json:"proof,omitempty"` // For Cortensor cryptographic proofs
}

// InferenceProvider is the core interface for interacting with LLM backends
type InferenceProvider interface {
	// Name returns the provider identifier
	Name() string

	// GetCompletion generates a response for the given request
	GetCompletion(ctx context.Context, req CompletionRequest) (CompletionResponse, error)

	// VerifyProof (Optional/Phase 4) verifies a cryptographic proof provided by the network
	VerifyProof(ctx context.Context, proof string) (bool, error)

	// ManageSession (Optional/Phase 2) handles provider-specific handshake/maintenance
	ManageSession(ctx context.Context) error
}
