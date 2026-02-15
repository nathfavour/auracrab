# Auracrab: Multi-Tenant Agentic Framework Roadmap

Transitioning from a monolithic agent to a multi-user, multi-agent operating framework.

## Phase 1: Identity & Isolation (Architectural Foundation)
- [ ] **Agent Identity Schema**: Define `Agent` struct in `pkg/identity` (ID, Handle, DisplayName, CreatedAt).
- [ ] **Global Registry**: Implement a global agent registry in `~/.auracrab/registry.json` to track all local agent accounts.
- [ ] **Path Namespacing**: Refactor `pkg/config` to support namespaced data directories: `~/.auracrab/agents/{agent_id}/`.
- [ ] **Context Propagation**: Update core functions to accept `context.Context` carrying the `AgentID`.
- [ ] **Migration Tool**: Create a script/function to migrate existing monolithic data to the `auracrab` (default) agent namespace.

## Phase 2: Multi-Butler Orchestration
- [ ] **Butler Refactoring**: De-singletonize `pkg/core/butler.go`. Enable instantiation with a specific `AgentID`.
- [ ] **Butler Manager**: Implement a manager to handle the lifecycle (Start/Stop/Status) of multiple agent instances concurrently.
- [ ] **Resource Isolation**: Ensure each `Butler` instance has its own:
    - Task Queue & Scheduler
    - Memory Stores (Vector & History)
    - Secret Vault
    - Social Bot Integrations
- [ ] **Shared Framework**: Ensure all agents leverage the same `pkg/crabs` (specialized agent skills) while maintaining private `Crab` registries.

## Phase 3: CLI & UX for Multi-Tenancy
- [ ] **Agent Management Commands**:
    - `auracrab agents create <handle>`: Provision a new clean-slate agent.
    - `auracrab agents list`: View all managed agents.
    - `auracrab agents delete <handle>`: Wipe an agent's data.
- [ ] **Context Switching**:
    - Implement a global `--agent <handle>` flag for all CLI commands.
    - `auracrab agents switch <handle>`: Set the default agent for the current session.
- [ ] **Multi-Agent TUI**: Update the TUI to allow switching between agent dashboards or viewing a consolidated view.

## Phase 4: Robustness & Scaling
- [ ] **Database Integration**: (Optional/Future) Transition from flat JSON files to an embedded DB (e.g., SQLite) for better multi-user performance.
- [ ] **Permission System**: Basic read/write permissions for shared resources between agents.
- [ ] **Framework SDK**: Refine `pkg/core` so third-party developers can easily "plug in" new agent logic while leveraging the Auracrab framework.
- [ ] **Millions-Scale POC**: Test the framework's ability to handle high volumes of idle/active agent identities.
