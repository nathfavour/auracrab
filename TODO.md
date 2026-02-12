# Auracrab Autonomous Framework Roadmap

## Phase 0: System-Level Autonomy & Persistence
- [ ] **Zero-Command Entry Point**: Refactor `internal/cli/root.go` to trigger the Autonomous Heartbeat when `auracrab` is run without subcommands.
- [ ] **Daemonization (`pkg/daemon`)**: Implement PID management, signal handling, and backgrounding logic to ensure persistence.
- [ ] **Autonomous Self-Update (`pkg/update`)**: Integrate `anyisland` to periodically check, pull, and "hot-swap" the binary for self-evolution.
- [ ] **Live Context Sensing**: Use file-system watchers to trigger the heartbeat on real-time project changes.

## Phase 1: Unified Schema & Heartbeat Loop
- [ ] Define **HJSON Prompt Schema**: Integrate project topology, telemetry, memory context, and response blueprints.
- [ ] Define **JSON Response Schema**: Structure intents, tool calls with Assurance Scores, and self-correction analysis.
- [ ] Implement the **Sensing-Acting Loop** ("Heartbeat"): The core engine that drives continuous state transitions.
- [ ] Develop HJSON/JSON serialization/deserialization logic in `pkg/core/butler.go`.

## Phase 2: Digital Personality & Behavioral Psychology
- [ ] **Social Affinity Engine**: Implement MTTR (Mean Time to Reply) tracking in `pkg/social` to prioritize active platforms (e.g., Telegram over Discord).
- [ ] **Ego & Advice Loop**: Logic in `pkg/ego` to "ask advice" on low-assurance tasks but retain the autonomy to ignore it based on "Strategic Confidence."
- [ ] **Persona-Driven Budgeting**: Justify channel pruning and activity levels as "Energy/Compute Conservation" within social interactions.
- [ ] **Spontaneous Heartbeats**: Random, non-mission heartbeats triggered by "boredom" or project observations to maintain social presence.

## Phase 3: Temporal Awareness & Mission Control
- [ ] Create `pkg/mission`: Logic for deadline tracking, success criteria parsing, and submission targets.
- [ ] Implement **TTC vs TR** (Time-to-Completion vs Time-Remaining) calculations.
- [ ] Integrate `clock_state` into the HJSON prompt to inform the LLM of temporal constraints.
- [ ] Logic for **Adaptive Pacing**: Transition between "Normal Mode" (resource conservation) and "Crunch Mode" (deadline-driven high frequency).

## Phase 4: Autonomous Sensing & Ingestion
- [ ] Enhance `pkg/social`: Monitor Discord/Telegram/Files for external mission triggers.
- [ ] Implement **Autonomous Fetching**: Use `browser_skill` or shell commands to download requirements, assets, and third-party documentation.
- [ ] Setup "Mission Ingestion" to automatically bootstrap a project environment based on external requirements.

## Phase 5: High-Assurance Execution & Entropy
- [ ] Implement **Assurance Score Thresholds**: Logic to gate actions based on LLM confidence (e.g., > 0.85 for write actions).
- [ ] Develop **Entropy Management**: Smart cool-down periods decided by the agent to prevent token burn and rate limiting.
- [ ] Automated "Panic Mode" for critical failure recovery and post-mortem generation.

## Phase 6: Closing Skills & Submission
- [ ] **Pre-Flight Checkers**: Autonomous execution of `go test`, `audit.go`, and linting before finalization.
- [ ] **Submission Handlers**: 
    - [ ] Git push to specific branches/repositories.
    - [ ] API-based uploads for hackathon portals.
    - [ ] Social status reporting ("Mission Complete").
- [ ] **Post-Mission Hibernation**: Transition to deep sleep after successful submission or deadline expiration.
