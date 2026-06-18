package sandbox

import "context"

// VerifyRequest carries sandbox verification parameters.
type VerifyRequest struct {
	WorkDir string
	Command string
	Image   string
}

// VerifyResult reports sandbox execution outcome.
type VerifyResult struct {
	Success  bool
	ExitCode int
	Output   string
}

// ExecutionInterface runs isolated compiler and test matrices.
type ExecutionInterface interface {
	Execute(ctx context.Context, req VerifyRequest) (*VerifyResult, error)
}
