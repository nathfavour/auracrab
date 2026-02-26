package spine

import (
	"context"
	"fmt"
	"time"

	"github.com/nathfavour/auracrab/pkg/biology"
)

// Cell represents a unit of work or a sub-agent that attaches to the spine.
type Cell interface {
	Pulse(ctx context.Context) error
	Name() string
}

// Spine is the central nervous system pulse.
type Spine struct {
	cells []Cell
	rate  time.Duration
}

func NewSpine(rate time.Duration) *Spine {
	return &Spine{
		cells: []Cell{},
		rate:  rate,
	}
}

func (s *Spine) Attach(cell Cell) {
	s.cells = append(s.cells, cell)
}

// Breathes starts the heartbeat loop with adaptive rate.
func (s *Spine) Breathes(ctx context.Context) {
	fmt.Printf("SPINE: Starting pulse. Initial rate %v\n", s.rate)
	
	for {
		// Calculate current metabolic rate
		energy, _ := biology.CheckThermodynamics()
		currentRate := s.rate

		// Adaptive logic: Slow down if energy is low
		if energy.EnergyLevel < 0.2 {
			currentRate = s.rate * 10 // Slow to 0.1Hz
		} else if energy.EnergyLevel < 0.5 {
			currentRate = s.rate * 2 // Slow to 0.5Hz
		}

		select {
		case <-ctx.Done():
			fmt.Println("SPINE: Context cancelled, stopping pulse.")
			return
		case <-time.After(currentRate):
			s.pulse(ctx)
		}
	}
}

func (s *Spine) pulse(ctx context.Context) {
	// 1. Check Physics (Thermodynamics & Entropy)
	energy, err := biology.CheckThermodynamics()
	if err != nil {
		fmt.Printf("SPINE ERROR: Failed to check thermodynamics: %v\n", err)
	} else {
		if biology.ShouldApoptose() {
			biology.Apoptosis("Critical energy depletion")
		}
		
		// Optional: Log energy state on pulse if it's significant
		if energy.EnergyLevel < 0.2 {
			fmt.Printf("SPINE WARNING: Low Energy Level: %.2f (CPU: %.1f%%, MEM: %.1f%%)\n", 
				energy.EnergyLevel, energy.CPUUsage, energy.MemoryUsage)
		}
	}

	// 2. Pulse all attached cells
	for _, cell := range s.cells {
		go func(c Cell) {
			err := c.Pulse(ctx)
			if err != nil {
				fmt.Printf("SPINE: Cell '%s' failed pulse: %v\n", c.Name(), err)
			}
		}(cell)
	}
}
