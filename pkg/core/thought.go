package core

import (
	"fmt"
	"time"

	"github.com/nathfavour/auracrab/pkg/biology"
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
	pulseFrame := fmt.Sprintf(
		"CURRENT_PULSE: #%d (Metabolic Rate: %.2fHz)\nENERGY: %.2f/1.00 (Burned: %.3f)\n",
		signature.PulseCount+1, 1.0, bio.EnergyLevel, burn,
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
	registry := m.butler.registry // Or specialized skill registry
	// For now using the global registry
	allSkills := []string{}
	// Fallback to all if none specified
	if len(active) == 0 {
		return "(No specific skills expressed. Use core logic.)"
	}

	for _, name := range active {
		// In a real implementation, we'd pull from the actual skill registry
		allSkills = append(allSkills, fmt.Sprintf("- skill: %s (expressed)", name))
	}
	return "ACTIVE_DNA:\n" + fmt.Join(allSkills, "\n")
}

func (m *Metabolizer) metabolizeFovea(fovea *Fovea) string {
	// Detailed content of foveated files
	return fmt.Sprintf("FOCUS_FILES: %v (Detailed in next pulse if needed)", fovea.Files)
}
