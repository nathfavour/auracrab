# Auracrab: Browser-Native Agent Framework (Web Use)

Implementing human-like, multi-tenant browser automation integrated directly into the Auracrab toolkit.

## Phase 1: Bridge & Connectivity (Completed)
- [x] **WebSocket Bridge**: Established `pkg/connect/browser.go` for real-time agent-to-extension communication.
- [x] **WXT Extension Skeleton**: Initialized `webuse/` extension with permissions for tab control and scripting.
- [x] **Basic Command Loop**: Support for `open` and `scrape` commands via WebSocket.
- [x] **Browser Skill Integration**: Refactored `pkg/skills/browser_skill.go` to leverage the extension when available.

## Phase 2: Multi-Session & Profile Intelligence (Completed)
- [x] **Profile Identification**: Enhance extension to report specific browser profiles, window IDs, and account contexts.
- [x] **Intelligent Client Registry**: Update `BrowserChannel` to track and address specific browser windows or profiles.
- [x] **Account-Aware Automation**: Ability for the agent to say "Use the browser profile where I'm logged into Twitter."
- [x] **Context Switching**: Allow the agent to multiplex commands across different browser instances.

## Phase 3: Human-Like Automation Toolkit
- [x] **Advanced Interaction Skill**: Implement JSON-based high-level actions:
    - `click(selector)`: Smooth scroll + natural click.
    - `type(selector, text)`: Simulated keystrokes with varied delays.
    - `hover(selector)`: Move "virtual" cursor.
    - `wait(condition)`: Intelligent waiting for DOM elements or network idle.
- [x] **Visual Context**: Enable extension to capture screenshots or DOM snapshots for the AI to "see" the page layout.
- [ ] **Intelligent URL Discovery**: Agent can perform Google searches or guess URLs based on service names if not provided.

## Phase 4: Full Tool Integration & Autonomy
- [x] **Zero-Headless Philosophy**: Ensure 100% of automation happens in the user's local session, avoiding bot detection and leveraging existing auth.
- [x] **Autonomous Browser Missions**: Support long-running tasks that involve navigating complex web workflows via the `browser_agent` skill.
- [ ] **Human-Agent Handover**: UI in the extension to allow the human to take over or provide input when the AI is stuck.
- [x] **Account Independent Treatment**: Ensure the AI treats different accounts of the same service (e.g., Personal vs Work Gmail) as distinct entities via context-awareness.

## Phase 5: Hardcode-Free Intelligence
- [x] **Dynamic Manifests**: No hardcoded selectors; AI should analyze the DOM (via `scrape_interactive`) to find interaction points.
- [x] **Self-Correcting Scripts**: If a click fails, the AI should retry with a different selector or strategy in its autonomous loop.
- [ ] **Toolkit-to-Browser Mapping**: Mapping all Auracrab skills (Vault, Memory, Crabs) so they can be triggered from or feed into browser automation.
