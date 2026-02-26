package core

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/nathfavour/auracrab/pkg/biology"
)

// ThoughtCell generates autonomous reflections and actions.
type ThoughtCell struct {
	butler *Butler
	lastThought time.Time
}

func NewThoughtCell(b *Butler) *ThoughtCell {
	return &ThoughtCell{
		butler: b,
	}
}

func (tc *ThoughtCell) Name() string {
	return "ThoughtGenerator"
}

func (tc *ThoughtCell) Pulse(ctx context.Context) error {
	// Only think if we have surplus energy (Thermodynamic Rule)
	energy, _ := biology.CheckThermodynamics()
	if energy.EnergyLevel < 0.6 {
		return nil // Conserve energy
	}

	// Think every 10-30 minutes randomly
	interval := time.Duration(10+rand.Intn(20)) * time.Minute
	if time.Since(tc.lastThought) < interval {
		return nil
	}

	tc.lastThought = time.Now()
	
	// Autonomous Reflection
	go tc.reflect(ctx)

	return nil
}

func (tc *ThoughtCell) reflect(ctx context.Context) {
	prompt := "SYSTEM_REFLEX: You are idling. Generate a brief, punchy autonomous reflection on your current environment or a random system optimization idea. Keep it under 140 characters."
	
	thought, err := tc.butler.QueryWithContext(ctx, prompt, "vibe")
	if err != nil {
		return
	}

	fmt.Printf("THOUGHT: %s\n", thought)
	
	// Broadcast to verbose channels/owners
	// tc.butler.BroadcastLog(thought)
}
