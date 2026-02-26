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
}

func NewImmuneSystem(nodeID string) *ImmuneSystem {
	regDir := filepath.Join(config.DataDir(), "swarm")
	_ = os.MkdirAll(regDir, 0755)

	return &ImmuneSystem{
		registry: regDir,
		self: &Node{
			PID:    os.Getpid(),
			ID:     nodeID,
			Status: StatusHealthy,
			BornAt: time.Now(),
		},
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

func (is *ImmuneSystem) voteToKill(n Node) {
	fmt.Printf("IMMUNE: Node %d is mutated/unstable. Casting vote for Apoptosis.\n", n.PID)
	
	// In a real Raft-like system, we'd count votes. 
	// For this local biological system, if we see a mutated node, we might send a SIGKILL 
	// if we are the "Alpha" or if there's consensus.
	// For now, let's just log it.
	
	// biology.Apoptosis is for SELF. To kill another, we use syscall.
	// process, _ := os.FindProcess(n.PID)
	// process.Signal(syscall.SIGKILL)
}

// Name implements spine.Cell
func (is *ImmuneSystem) Name() string {
	return "ImmuneSystem"
}

// Pulse implements spine.Cell
func (is *ImmuneSystem) Pulse(ctx context.Context) error {
	return is.Ping(ctx)
}
