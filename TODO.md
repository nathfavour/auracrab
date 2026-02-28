# 🦀 AURACRAB: THE BIOLOGICAL SYSTEMS ROADMAP

This roadmap tracks the evolution of Auracrab into a self-sustaining, resilient, and autonomous biological entity.

## Current Focus: Phase 4 & 5
Our current objective is to deepen the metabolic resilience of Auracrab and begin the transition to a distributed swarm.

### Phase 4: Metabolism & Hibernation
- [ ] **Adaptive Metabolism**: Refine variable heartbeat (Spine rate) logic (1Hz -> 0.1Hz) based on fine-grained energy/activity telemetry.
- [ ] **Hibernation State**: Implement deep sleep (0.01Hz) triggered by inactivity or specific time-based cycles.
- [ ] **Automated Apoptosis**: Implement systemic cleanup of "dead cells" (temp files, stale clones, old logs) during idle/hibernation cycles.

### Phase 5: Swarm Consensus (Distributed Immune System)
- [ ] **Isolated I/O Band**: Secure, encrypted node-to-node channels for swarm coordination (Moved from Phase 3).
- [ ] **Node Hand-off**: Ability for an overloaded node to delegate metabolic load to a peer.
- [ ] **Consensus Voting**: Swarm-wide voting to terminate "mutated" or runaway processes.
- [ ] **Autonomous Cloning**: Self-replication triggered by high systemic demand and healthy energy reserves.

### Phase 6: Metabolic Optimization (The Path of Least Resistance)
- [ ] **Tiered Cognition**: Heuristic gate to solve tasks with local tools (bash/grep) before escalating to expensive AI tokens.
- [ ] **Semantic Habituation**: Cache successful `PulsePlans` to reuse them for similar future goals without re-planning.
- [ ] **Foveated Perception**: Dynamically prune context windows to only "see" what is essential for the current atomic step.

---

## Completed Milestones

### Phase 1: Proprioception (Internal Sense)
- [x] **Internal State Injection**: Feed `Energy` levels (CPU/RAM) and `Node Health` directly into the System Prompt.
- [x] **Metabolism Tracking**: Implement an "Energy Cost" for every API call and compute-intensive action.
- [x] **Thermodynamic Reasoning**: Enable the AI to postpone or abort tasks based on energy thresholds.

### Phase 2: The Task Nervous System (Asynchronous Granularity)
- [x] **Recursive Task Tree**: Break high-level user goals into atomic, schedulable `Pulse` actions.
- [x] **Spine-Integrated Execution**: Tasks no longer block; they attach to the Spine and execute over multiple heartbeats.
- [x] **Progress Pulsing**: Automatically drop "Lazy I/O" updates to the operator as sub-tasks complete.

### Phase 3: Communication Bands (I/O Architecture)
- [x] **Fast I/O Band**: Immediate, high-priority interrupts for operator commands.
- [x] **Lazy I/O Band**: Throttled background state synchronization and status updates via Spine Pulse.
