package spine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nathfavour/auracrab/pkg/biology"
)

// Cell represents a unit of work or a sub-agent that attaches to the spine.
type Cell interface {
	Pulse(ctx context.Context) error
	Name() string
}

type Event struct {
	Type      string
	Payload   interface{}
	Timestamp time.Time
}

type EventHandler interface {
	Handle(e Event)
}

// Spine is the central nervous system pulse.
type Spine struct {
	mu       sync.RWMutex
	cells    []Cell
	handlers []EventHandler
	rate     time.Duration
	energy   biology.Energy
}

func NewSpine(rate time.Duration) *Spine {
	s := &Spine{
		cells:    []Cell{},
		handlers: []EventHandler{},
		rate:     rate,
	}
	// Initial energy check
	s.energy, _ = biology.CheckThermodynamics()
	// Start background energy sensor
	go s.sense(context.Background())
	return s
}

func (s *Spine) Attach(cell Cell) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cells = append(s.cells, cell)
}

func (s *Spine) RegisterHandler(h EventHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers = append(s.handlers, h)
}

func (s *Spine) Broadcast(e Event) {
	s.mu.RLock()
	handlers := make([]EventHandler, len(s.handlers))
	copy(handlers, s.handlers)
	s.mu.RUnlock()

	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}

	for _, h := range handlers {
		go h.Handle(e)
	}
}

// sense periodically updates the spine's awareness of its physical state.
func (s *Spine) sense(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			energy, err := biology.CheckThermodynamics()
			if err == nil {
				s.mu.Lock()
				s.energy = energy
				s.mu.Unlock()
			}
		}
	}
}

// Breathes starts the heartbeat loop with adaptive rate.
func (s *Spine) Breathes(ctx context.Context) {
	fmt.Printf("SPINE: Starting pulse. Initial rate %v\n", s.rate)

	for {
		s.mu.RLock()
		energy := s.energy
		s.mu.RUnlock()

		currentRate := s.rate
		metabolism := biology.GetMetabolism()
		_, uptime := metabolism.GetStats()

		// 1. Activity-based Hibernation (0.01Hz)
		// If no activity for 5 minutes, enter deep sleep.
		idleTime := time.Since(metabolism.LastActivity)
		if uptime > 1*time.Minute && idleTime > 5*time.Minute {
			currentRate = s.rate * 100 // Slow to 0.01Hz (100s)
		} else if energy.EnergyLevel < 0.2 {
			// 2. Energy-based adaptation (0.1Hz)
			currentRate = s.rate * 10 // Slow to 0.1Hz (10s)
		} else if energy.EnergyLevel < 0.5 {
			// 3. Mild fatigue adaptation (0.5Hz)
			currentRate = s.rate * 2 // Slow to 0.5Hz (2s)
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
	s.mu.RLock()
	energy := s.energy
	s.mu.RUnlock()

	// 1. Check Physics (Thermodynamics & Entropy)
	if biology.ShouldApoptose() {
		biology.Apoptosis("Critical energy depletion")
	}

	// Optional: Log energy state on pulse if it's significant
	if energy.EnergyLevel < 0.2 {
		fmt.Printf("SPINE WARNING: Low Energy Level: %.2f (CPU: %.1f%%, MEM: %.1f%%)\n",
			energy.EnergyLevel, energy.CPUUsage, energy.MemoryUsage)
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
