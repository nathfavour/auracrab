package core

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/nathfavour/auracrab/pkg/biology"
	"github.com/nathfavour/auracrab/pkg/skills"
)

// ThoughtSignature represents a distilled state of a multi-pulse goal.
type ThoughtSignature struct {
	PulseCount     int
	Goal           string
	LastResult     string
	RemainingSteps []string
	Anomalies      []string
}

func (ts *ThoughtSignature) String() string {
	if ts == nil {
		return "SIGNATURE: (New metabolic goal initialized)"
	}
	return fmt.Sprintf(
		"PULSE_SIGNATURE [#%d]:\nGoal: %s\nLast Progress: %s\nRemaining Steps: %v\nAnomalies: %v",
		ts.PulseCount, ts.Goal, ts.LastResult, ts.RemainingSteps, ts.Anomalies,
	)
}

// Metabolizer assembles the dynamic, foveated prompt.
type Metabolizer struct {
	butler *Butler
}

func NewMetabolizer(b *Butler) *Metabolizer {
	return &Metabolizer{butler: b}
}

// Fovea defines the high-detail focus area for the current pulse.
type Fovea struct {
	Files        []string // Detailed contents of these files
	WorkingDir   string
	ActiveSkills []string // Only these skills will be expressed
}

func (m *Metabolizer) Build(
	userPrompt string,
	signature *ThoughtSignature,
	fovea *Fovea,
) string {
	// 1. Biological Proprioception
	bio, _ := biology.CheckThermodynamics()
	met := biology.GetMetabolism()
	burn, _ := met.GetStats()

	// 2. Skill Sharding (DNA Expression)
	skillDNA := m.metabolizeSkills(fovea.ActiveSkills)

	// 3. Foveated Sensing (Focus Area)
	contextDNA := m.metabolizeFovea(fovea)

	// 4. Temporal Pulse Framing
	pulseCount := 0
	if signature != nil {
		pulseCount = signature.PulseCount
	}
	pulseFrame := fmt.Sprintf(
		"CURRENT_PULSE: #%d (Metabolic Rate: %.2fHz)\nENERGY: %.2f/1.00 (Burned: %.3f)\n",
		pulseCount+1, 1.0, bio.EnergyLevel, burn,
	)

	return fmt.Sprintf(
		"AURACRAB_LIVING_PROMPT\n\n"+
			"== TEMPORAL_FRAME ==\n%s\n"+
			"== BIOLOGICAL_STATE ==\n"+
			"- CPU: %.1f%% | MEM: %.1f%%\n"+
			"- THERMODYNAMIC_LIMIT: 0.15\n\n"+
			"== THOUGHT_SIGNATURE ==\n%s\n\n"+
			"== FOVEATED_SENSING (HIGH_DETAIL) ==\n%s\n\n"+
			"== SKILL_EXPRESSION (DNA) ==\n%s\n\n"+
			"== USER_PULSE_REQUEST ==\n%s\n\n"+
			"== METABOLIC_DIRECTIVE ==\n"+
			"- Act within the current temporal window.\n"+
			"- Distill the result for the next pulse signature.\n"+
			"- Prioritize energy efficiency. Abort if energy < 0.15.\n"+
			"- Output final result or next required action only.",
		pulseFrame,
		bio.CPUUsage, bio.MemoryUsage,
		signature.String(),
		contextDNA,
		skillDNA,
		userPrompt,
	)
}

func (m *Metabolizer) metabolizeSkills(active []string) string {
	reg := skills.GetRegistry()
	var expressed []string

	if len(active) == 0 {
		return "ACTIVE_DNA: (General reasoning, no specialized skills expressed)"
	}

	for _, name := range active {
		if s, ok := reg.Get(name); ok {
			expressed = append(expressed, fmt.Sprintf("- %s: %s\n  manifest: %s", s.Name(), s.Description(), string(s.Manifest())))
		}
	}

	return "ACTIVE_DNA:\n" + strings.Join(expressed, "\n")
}

func (m *Metabolizer) metabolizeFovea(fovea *Fovea) string {
	if fovea == nil || len(fovea.Files) == 0 {
		return "FOVEA: (General context, no specific file focus)"
	}
	// In a real implementation, we'd read_file for each and include content
	return fmt.Sprintf("FOCUS_FILES: %v", fovea.Files)
}

// ReflexCell generates autonomous reflections and actions.
type ReflexCell struct {
	butler      *Butler
	lastThought time.Time
}

func NewReflexCell(b *Butler) *ReflexCell {
	return &ReflexCell{
		butler: b,
	}
}

func (rc *ReflexCell) Name() string {
	return "ReflexGenerator"
}

func (rc *ReflexCell) Pulse(ctx context.Context) error {
	energy, _ := biology.CheckThermodynamics()
	if energy.EnergyLevel < 0.6 {
		return nil
	}

	interval := time.Duration(10+rand.Intn(20)) * time.Minute
	if time.Since(rc.lastThought) < interval {
		return nil
	}

	rc.lastThought = time.Now()
	go rc.reflect(ctx)
	return nil
}

func (rc *ReflexCell) reflect(ctx context.Context) {
	prompt := "SYSTEM_REFLEX: You are idling. Generate a brief, punchy autonomous reflection on your current environment. Keep it under 140 characters."
	thought, err := rc.butler.QueryWithContext(ctx, prompt, "vibe")
	if err != nil {
		return
	}
	fmt.Printf("REFLEX: %s\n", thought)
}
