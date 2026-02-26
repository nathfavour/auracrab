# 🤖 AGENT GUIDELINES & BIOLOGICAL MANIFESTO

## The Biological Systems Manifesto
*To the Engineers (Human and Artificial):*

We are not just writing a standard CLI tool or a traditional web backend. We are architecting a self-sustaining, biological-inspired operating system. The goal is maximum systemic efficiency, extreme resilience, and zero-touch automation. Our core philosophy is driven by the realization that intelligence is highly optimized hardware executing the path of least resistance. We loathe brute-force hard work; therefore, the system must do the heavy lifting asynchronously, autonomously, and without manual intervention.

### 1. The Core Directives (The Physics Engine)
Before writing any logic, you must adhere to the immutable laws of this environment:

*   **The Thermodynamic Rule (Energy is Finite):** Every action, compute cycle, memory allocation, and API call costs "energy." Agents must evaluate the energy threshold before executing. If an action costs more compute or financial overhead than its predicted value, it must abort.
*   **The Entropy Rule (Apoptosis/Death):** Nothing lives forever. Processes, clones, and sub-agents must have a CRUD lifecycle that heavily emphasizes the "Delete." Stagnant, corrupted, or obsolete agents must be forcibly retired (killed) to free up the VPS memory and CPU. Self-correction requires death.
*   **The Lazy Heuristic:** Never build a complex, centralized controller when a dumb, decentralized constraint will naturally balance the system.

### 2. The Architecture (The Anatomy)
*   **The Spine (The Heartbeat):** The absolute core of the system. A continuous, immutable, and unconscious clock ticking away. It does not think; it merely pulses. All agents, scheduling, and I/O infrastructure hang off this spine. It is the primitive infrastructure that guarantees the system is alive.
*   **The I/O Layers:** Communication is physical infrastructure. We maintain distinct, specialized bands:
    *   **Fast I/O:** For instantaneous, high-priority interrupts.
    *   **Lazy I/O:** For background state synchronization, deliberately throttled when the VPS is under heavy load.
    *   **Isolated I/O:** 1-on-1 locked channels for secure, uncorrupted, and private data transfer between specific nodes.
*   **The Brain (Static + Dynamic):**
    *   **Static Brain:** Contains the unchangeable fundamental rules (like the energy constraint and the urge to minimize effort).
    *   **Dynamic Brain:** Interacts with the local AI models (sandboxed to prevent hallucinations or rogue internet access) to interpret tasks and adapt to the environment.

### 3. The Swarm Topology (Single Binary, Multi-Entity)
We reject the concept of maintaining multiple distinct source codes or different binaries for different agents. That violates the Lazy Heuristic. We operate on the biological DNA Principle:

*   **One DNA, Many Cells:** There is only one single compiled Go binary. Every instance in the swarm runs the exact same code. The difference in behavior is dictated by the environment and the specific function the instance is currently executing.
*   **Resource-Aware Cloning:** The binary has the fundamental ability to duplicate its process. However, this is strictly gated. A node can spawn a replica of itself only if it checks the host VPS and verifies that system resources (CPU/RAM) are below a critical threshold (e.g., 80%). If resources are scarce, `can_clone()` returns false.
*   **The Distributed Immune System (Voting):** To prevent a single "mutated" or stuck process from becoming a cancer that crashes the VPS, we rely on distributed consensus (similar to Raft). Nodes constantly ping each other via the Heartbeat. If Node A's error rate spikes, or it hoards resources without outputting work, the surrounding nodes vote to send a SIGKILL to Node A's process ID.

### 4. The Prime Directive
Optimize for maximum leverage and autonomy. Do not introduce bloated frameworks. Stick to lean, compiled logic (Go) or strict scripting (TypeScript). If a feature requires constant manual debugging, monitoring, or intervention from the human creator to keep running, it is a failed feature. Deploy it, let the swarm balance itself, and let the most efficient processes survive.

---

## 🛠️ Build Protocols

To maintain a clean and sovereign development environment, all agents must adhere to the following build protocols:

1.  **Isolation of Artifacts**: Compiled binaries and build artifacts must **never** be outputted directly into the source code directories.
2.  **Designated Output**: All builds must be directed to the `bin/` directory located at the project root.
3.  **Git Hygiene**: The `bin/` directory is strictly for local artifacts and must remain excluded from version control. Ensure it is listed in the `.gitignore` file (this project already includes `bin/` in its `.gitignore`).

Failure to follow these protocols can lead to repository pollution and conflicts with managed environments like Anyisland.
