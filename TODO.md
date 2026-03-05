Auracrab Autonomy Upgrade - Planning and Orchestration TODO

Phase 0 - Baseline and Constraints
- Confirm current runtime entrypoints and service modes (CLI/TUI/daemon)
- Inventory active channels (Telegram, Discord, CLI) and task intake flows
- Map existing persistence points (tasks JSON, missions JSON, history SQLite)
- Document current task lifecycle states and timeouts
- Identify any non-goal areas that must remain untouched

Phase 1 - Continuous Execution Spine Integration
- Add a long-running pulse loop to Butler serve lifecycle
- Define pulse cadence and backoff parameters
- Attach NervousSystem cell to Spine
- Attach ImmuneSystem cell to Spine
- Attach ReflexCell (if enabled) to Spine
- Ensure graceful shutdown of pulse loop on context cancel
- Add health logging for each pulse cycle

Phase 2 - Task Continuity JSON Schema
- Define JSON schema version and compatibility rules
- Specify core task fields (goal, status, priority, timestamps)
- Specify plan structure (steps, dependencies, weights)
- Specify signature fields (pulse_count, remaining_steps, anomalies)
- Specify memory fields (habituation_key, history_refs, vector_refs)
- Specify meta fields (retry_count, energy_budget, last_error)
- Create schema validation rules and defaults
- Define migration path for existing tasks JSON

Phase 3 - Butler Task Model Upgrade
- Expand Task struct to include plan/steps and cursor
- Add checkpoint timestamps to task state
- Persist full task continuity JSON on every step transition
- Load and hydrate tasks with step state on startup
- Resume running or paused tasks on boot
- Add task retry policy with max retries and backoff
- Add task abandonment policy for stuck tasks

Phase 4 - NervousSystem Execution Loop
- Map plan steps to PulseTask execution
- Enforce step dependencies and weights
- Implement step result capture and persistence
- Update ThoughtSignature from step outcomes
- Implement per-step energy budget checks
- Implement resume logic from last checkpoint
- Add failure handling and fallback pathways

Phase 5 - Mission DAG Orchestration
- Convert Mission tasks into plan steps with depends_on
- Execute Mission.GetExecutableTasks() on each pulse
- Persist Mission execution state in continuity JSON
- Add mission-level status updates (running, blocked, completed)
- Reconcile mission completion with Butler task status

Phase 6 - History and Memory Continuity
- Write task events to history store (start, step, finish, failure)
- Link task records to conversation history IDs
- Cache normalized goal for habituation
- Define retention policy for historical task events

Phase 7 - Skill Execution and Payload Contracts
- Standardize step input payload shape for skills registry
- Validate skill args as JSON before execution
- Add structured output capture (string or JSON result)
- Define error contract for skill failures

Phase 8 - Safety, Entropy, and Kill-Switches
- Define energy budget thresholds per task and per pulse
- Implement task timeout and decay rules
- Add watchdog for tasks stuck in running state
- Add automatic cleanup of abandoned or invalid tasks
- Ensure immune system can flag and stop misbehaving tasks

Phase 9 - Observability and Debugging
- Add pulse-level logs with task summaries
- Add step-level logs with durations and outcomes
- Add task status metrics (pending, running, failed)
- Add trace IDs linking task, step, and history

Phase 10 - Backward Compatibility and Migration
- Create migration for existing tasks.json
- Create migration for missions.json if needed
- Test restart with mixed old/new task formats
- Provide rollback plan if migration fails

Phase 11 - Validation and Test Scenarios
- Add unit tests for schema validation
- Add tests for resume from checkpoint
- Add tests for mission DAG execution order
- Add tests for failure and retry policies
- Add tests for spine shutdown cleanup

Phase 12 - Documentation and Operator Guidance
- Document new continuity schema
- Document pulse loop configuration
- Document task lifecycle and statuses
- Provide runbook for recovering stuck tasks
