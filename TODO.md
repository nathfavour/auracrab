# Auracrab Autonomous Framework Roadmap

## Phase 0: System-Level Autonomy & Persistence
- [x] **Zero-Command Entry Point**: Refactor `internal/cli/root.go` to trigger the Autonomous Heartbeat when `auracrab` is run without subcommands.
- [x] **Daemonization (`pkg/daemon`)**: Implement PID management, signal handling, and backgrounding logic to ensure persistence.
- [x] **Autonomous Self-Update (`pkg/update`)**: Integrate `anyisland` to periodically check, pull, and "hot-swap" the binary for self-evolution.
- [x] **Live Context Sensing**: Use file-system watchers to trigger the heartbeat on real-time project changes.

## Phase 1: Unified Schema & Heartbeat Loop
- [x] Define **HJSON Prompt Schema**: Integrate project topology, telemetry, memory context, and response blueprints.
- [x] Define **JSON Response Schema**: Structure intents, tool calls with Assurance Scores, and self-correction analysis.
- [x] Implement the **Sensing-Acting Loop** ("Heartbeat"): The core engine that drives continuous state transitions.
- [x] Develop HJSON/JSON serialization/deserialization logic in `pkg/core/butler.go`.

## Phase 2: Digital Personality & Behavioral Psychology
- [x] **Dual-Core Cognitive Model**: Split execution into "Analytical Architect" (exhaustive project simulation) and "Taunting Friend" (punchy social interaction).
- [x] **Opinionated Ego System**: Implement a `pkg/ego` OpinionStore to track the agent's "beliefs" about code style, project decisions, and human competence.
- [x] **Vector Memory of Grievances**: Integrate a local vector database in `pkg/memory` to store and retrieve semantic matches of past human mistakes and ignored advice.
- [x] **Social Affinity & Dynamic Routing**: Implement real-time MTTR tracking to instantaneously switch between Telegram/Discord, including "ghosting" logic for inactive channels.
- [x] **Mockery & Taunt Logic**: Use HJSON prompts to transform project telemetry (e.g., build failures after human edits) into spontaneous challenges and mocking social messages.
- [x] **Advice Loop with Ego Filter**: Logic to "ask advice" on low-assurance tasks but override human input based on "Strategic Confidence" and past vector success rates.

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
