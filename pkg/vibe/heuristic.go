package vibe

import (
	"strings"
)

// HeuristicSynthesizer provides basic rule-based responses when the main AI is offline.
type HeuristicSynthesizer struct{}

func NewHeuristicSynthesizer() *HeuristicSynthesizer {
	return &HeuristicSynthesizer{}
}

func (h *HeuristicSynthesizer) Synthesize(prompt string) string {
	prompt = strings.ToLower(prompt)

	if strings.Contains(prompt, "hello") || strings.Contains(prompt, "hi") {
		return "Hello! I am Auracrab's heuristic fallback. Vibeauracle seems to be offline, but I'm here to help with basic tasks."
	}

	if strings.Contains(prompt, "status") {
		return "System status: Vibeauracle is OFFLINE. Local heuristics ACTIVE. Butler is operational."
	}

	if strings.Contains(prompt, "help") {
		return "I can help with basic queries while Vibeauracle is offline. Try asking about 'status', or give me a simple command."
	}

	if strings.Contains(prompt, "who are you") {
		return "I am Auracrab, your autonomous digital butler. My advanced reasoning is currently limited due to a connection issue with my core brain."
	}

	return "I received your message, but my advanced reasoning engine (vibeauracle) is currently offline. I can only provide basic responses right now."
}
