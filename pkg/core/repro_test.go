package core

import (
	"testing"
)

func TestMetabolizerBuild_NilFovea(t *testing.T) {
	m := &Metabolizer{}
	// This should not panic
	_ = m.Build("test prompt", nil, nil)
}

func TestButlerQueryWithContext_NilFovea(t *testing.T) {
	// We might need to mock or initialize things for Butler
	// but let's see if we can just test the Metabolizer part which is where the panic is.
	b := &Butler{
		Metabolizer: &Metabolizer{},
	}
	
	// QueryWithContext calls QueryMetabolic which calls Build with nil fovea
	// We don't want it to actually call VibeClient during test if possible, 
	// but the panic happens BEFORE the client call.
	
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Recovered from panic: %v", r)
		}
	}()

	// Build is called first in QueryMetabolic
	b.Metabolizer.Build("test", nil, nil)
}
