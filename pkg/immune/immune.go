package immune

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/nathfavour/auracrab/pkg/biology"
	"github.com/nathfavour/auracrab/pkg/config"
)

type NodeStatus string

const (
	StatusHealthy NodeStatus = "healthy"
	StatusDegraded NodeStatus = "degraded"
	StatusMutated  NodeStatus = "mutated"
)

type Node struct {
	PID       int        `json:"pid"`
	ID        string     `json:"id"`
	Status    NodeStatus `json:"status"`
	ErrorRate float64    `json:"error_rate"`
	LastPing  time.Time  `json:"last_ping"`
	BornAt    time.Time  `json:"born_at"`
}

type ImmuneSystem struct {
	mu       sync.RWMutex
	registry string
	self     *Node
	bus      *SwarmBus
}

func NewImmuneSystem(nodeID string) *ImmuneSystem {
	regDir := filepath.Join(config.DataDir(), "swarm")
	_ = os.MkdirAll(regDir, 0755)

	selfPID := os.Getpid()
	return &ImmuneSystem{
		registry: regDir,
		self: &Node{
			PID:    selfPID,
			ID:     nodeID,
			Status: StatusHealthy,
			BornAt: time.Now(),
		},
		bus: NewSwarmBus(selfPID),
	}
}

// Register adds the current node to the swarm registry.
func (is *ImmuneSystem) Register() error {
	is.self.LastPing = time.Now()
	return is.writeSelf()
}

func (is *ImmuneSystem) writeSelf() error {
	path := filepath.Join(is.registry, fmt.Sprintf("node_%d.json", is.self.PID))
	data, err := json.Marshal(is.self)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Ping updates the node's presence and checks on neighbors.
func (is *ImmuneSystem) Ping(ctx context.Context) error {
	is.mu.Lock()
	is.self.LastPing = time.Now()
	// Update dynamic stats
	cpu, _, _ := biology.GetProcessStats()
	if cpu > 90.0 {
		is.self.Status = StatusDegraded
	} else {
		is.self.Status = StatusHealthy
	}
	is.mu.Unlock()

	if err := is.writeSelf(); err != nil {
		return err
	}

	return is.surveillance()
}

// surveillance looks for "diseased" or dead nodes.
func (is *ImmuneSystem) surveillance() error {
	files, err := filepath.Glob(filepath.Join(is.registry, "node_*.json"))
	if err != nil {
		return err
	}

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}

		var n Node
		if err := json.Unmarshal(data, &n); err != nil {
			continue
		}

		if n.PID == is.self.PID {
			continue
		}

		// 1. Check for Dead Nodes (Heartbeat failure)
		if time.Since(n.LastPing) > 30*time.Second {
			fmt.Printf("IMMUNE: Node %d is dead (ping timeout). Removing from registry.\n", n.PID)
			_ = os.Remove(f)
			continue
		}

		// 2. Check for Mutated Nodes (Voting)
		if n.Status == StatusMutated || n.ErrorRate > 0.5 {
			is.voteToKill(n)
		}
	}

	return nil
}

func (is *ImmuneSystem) handleSwarmMessage(msg SwarmMessage) {
	switch msg.Type {
	case "HANDOFF_REQUEST":
		fmt.Printf("IMMUNE: Received handoff request from node %d. Checking capacity...\n", msg.From)
		// If we are healthy and have capacity, we could "accept" it.
	case "VOTE_APOPTOSIS":
		targetPID := int(msg.Payload.(float64)) // JSON unmarshal makes it float64
		fmt.Printf("IMMUNE: Consensus vote for apoptosis of node %d received from node %d.\n", targetPID, msg.From)
	}
}

func (is *ImmuneSystem) voteToKill(n Node) {
	fmt.Printf("IMMUNE: Node %d is mutated/unstable. Casting broadcast vote for Apoptosis.\n", n.PID)
	_ = is.bus.Broadcast("VOTE_APOPTOSIS", n.PID)
}

// RequestHandoff broadcasts a request for another node to take over some load.
func (is *ImmuneSystem) RequestHandoff() {
	fmt.Println("IMMUNE: System overloaded. Requesting node hand-off...")
	_ = is.bus.Broadcast("HANDOFF_REQUEST", "High metabolic load detected")
}

// Name implements spine.Cell
func (is *ImmuneSystem) Name() string {
	return "ImmuneSystem"
}

// Pulse implements spine.Cell
func (is *ImmuneSystem) Pulse(ctx context.Context) error {
	// 1. Regular Ping and Surveillance
	if err := is.Ping(ctx); err != nil {
		return err
	}

	// 2. Handle Swarm Messages (Isolated I/O Band)
	messages, err := is.bus.Listen()
	if err == nil {
		for _, msg := range messages {
			is.handleSwarmMessage(msg)
		}
	}

	// 3. Automated Apoptosis (Systemic Cleanup)
	metabolism := biology.GetMetabolism()
	idleTime := time.Since(metabolism.LastActivity)

	// 4. Autonomous Cloning: Self-replication triggered by high systemic demand
	if biology.CanClone() && idleTime < 10*time.Second {
		// Only clone if we are actually busy and healthy.
		fmt.Println("IMMUNE: System under load but healthy. Triggering autonomous cloning...")
		_ = biology.Clone()
	}

	// If idle for more than 2 minutes, perform systemic cleanup
	if idleTime > 2*time.Minute {
		is.cleanup()
	}

	return nil
}

func (is *ImmuneSystem) cleanup() {
	fmt.Println("IMMUNE: Initiating systemic cleanup (Automated Apoptosis)...")

	// 1. Cleanup Temp Files
	tmpDir := filepath.Join(config.DataDir(), "tmp")
	_ = os.RemoveAll(tmpDir)
	_ = os.MkdirAll(tmpDir, 0755)

	// 2. Cleanup Old Logs (if applicable)
	// Assuming logs are in config.DataDir()/logs
	logDir := filepath.Join(config.DataDir(), "logs")
	files, err := os.ReadDir(logDir)
	if err == nil {
		for _, f := range files {
			info, err := f.Info()
			if err == nil && time.Since(info.ModTime()) > 7*24*time.Hour {
				_ = os.Remove(filepath.Join(logDir, f.Name()))
			}
		}
	}

	// 3. Cleanup Stale Registry Entries (Already handled by surveillance, but ensuring)
	is.surveillance()
}

