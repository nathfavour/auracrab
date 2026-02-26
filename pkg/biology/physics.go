package biology

import (
	"fmt"
	"os"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/process"
)

// Energy represents the available resources for the agent system.
type Energy struct {
	CPUUsage    float64 // Percentage
	MemoryUsage float64 // Percentage
	EnergyLevel float64 // 0.0 to 1.0 (1.0 is full health)
}

const (
	MaxCPUTreshold    = 80.0
	MaxMemoryTreshold = 85.0
	ApoptosisTreshold = 0.05 // Level at which the process should self-terminate
)

// CheckThermodynamics evaluates the system's energy state.
func CheckThermodynamics() (Energy, error) {
	c, err := cpu.Percent(time.Second, false)
	if err != nil {
		return Energy{}, err
	}

	v, err := mem.VirtualMemory()
	if err != nil {
		return Energy{}, err
	}

	cpuUsed := c[0]
	memUsed := v.UsedPercent

	// Simple heuristic: Energy decreases as usage increases
	cpuFactor := (100.0 - cpuUsed) / 100.0
	memFactor := (100.0 - memUsed) / 100.0
	
	level := (cpuFactor + memFactor) / 2.0

	return Energy{
		CPUUsage:    cpuUsed,
		MemoryUsage: memUsed,
		EnergyLevel: level,
	}, nil
}

// CanClone checks if the system has enough energy to spawn a replica.
func CanClone() bool {
	e, err := CheckThermodynamics()
	if err != nil {
		return false
	}
	// Strict gate: Must have low resource usage to clone
	return e.CPUUsage < MaxCPUTreshold && e.MemoryUsage < MaxMemoryTreshold
}

// ShouldApoptose checks if the process is "diseased" (corrupted or resource-starved).
func ShouldApoptose() bool {
	e, err := CheckThermodynamics()
	if err != nil {
		return false
	}

	// If energy is critically low for too long, apoptosis is triggered.
	if e.EnergyLevel < ApoptosisTreshold {
		return true
	}

	return false
}

// Apoptosis gracefully or forcibly retires the process.
func Apoptosis(reason string) {
	fmt.Printf("APOPTOSIS: %s. Releasing resources and exiting.\n", reason)
	// Log the death
	// In a real system, we might want to notify the swarm before dying.
	os.Exit(1)
}

// DNA returns the current process's executable path, representing its "genetic code".
func DNA() (string, error) {
	return os.Executable()
}

// Clone creates a new instance of the same binary.
func Clone() error {
	if !CanClone() {
		return fmt.Errorf("thermodynamic failure: insufficient energy to clone")
	}

	exe, err := DNA()
	if err != nil {
		return err
	}

	// Spawn a new process with same DNA
	attr := &os.ProcAttr{
		Dir:   ".",
		Env:   os.Environ(),
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	}

	fmt.Println("CLONING: Spawning a new cell...")
	p, err := os.StartProcess(exe, []string{exe, "serve", "--child"}, attr)
	if err != nil {
		return err
	}

	fmt.Printf("CLONING: New cell spawned with PID %d\n", p.Pid)
	return nil
}

// GetProcessStats returns stats for the current process.
func GetProcessStats() (float64, uint64, error) {
	p, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return 0, 0, err
	}

	cpuPercent, _ := p.CPUPercent()
	memInfo, _ := p.MemoryInfo()
	
	return cpuPercent, memInfo.RSS, nil
}
