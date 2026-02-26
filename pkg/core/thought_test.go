package core

import (
	"testing"
)

func TestMetabolizerBuild_NilFovea(t *testing.T) {
	m := &Metabolizer{}
	
	// Test with nil signature and nil fovea
	res := m.Build("test prompt", nil, nil)
	if res == "" {
		t.Error("Expected non-empty prompt")
	}

	// Test with non-nil signature and nil fovea
	sig := &ThoughtSignature{PulseCount: 1}
	res = m.Build("test prompt", sig, nil)
	if res == "" {
		t.Error("Expected non-empty prompt")
	}
}

func TestMetabolizerBuild_EmptyFovea(t *testing.T) {
	m := &Metabolizer{}
	f := &Fovea{}
	res := m.Build("test prompt", nil, f)
	if res == "" {
		t.Error("Expected non-empty prompt")
	}
}
