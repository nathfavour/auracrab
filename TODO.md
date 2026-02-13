# Auracrab Recovery & Stabilization Roadmap

## Immediate Fixes: Connectivity & Resilience
- [ ] **Remove Hardcoded IPC Timeouts**: Strip the 30s `SetReadDeadline` in `pkg/vibe/client.go` to support slow LLM providers.
- [ ] **Implement Heuristic Fallback**: Add a local `HeuristicSynthesizer` (ported from anyisland) to allow basic functionality when `vibeauracle` is offline.
- [ ] **Fix Empty Response Logic**: Adjust `CleanResponse` handling in `vibeauracle` or `auracrab` to ensure reasoning/thoughts are surfaced if the final content block is empty.

## Architectural Overhaul: Butler Queue & Prioritization
- [ ] **Asynchronous Message Spooler**: Refactor `Butler.handleChannelMessage` to use a non-blocking queue.
- [ ] **Message Prioritization**: Logic to prioritize "Critical System Alerts" and "Direct Tasks" over "General Chat."
- [ ] **Task Heartbeat**: Decouple the Telegram/Discord reply loop from the AI processing loop so the bot can send "Thinking..." or "Working..." status updates.

## Protocol & UX Improvements
- [ ] **IPC Streaming Support**: Upgrade UDS communication to handle streaming chunks (NDJSON) for real-time TUI/Bot feedback.
- [ ] **Context-Aware Intents**: Automatically switch between `IntentChat` (for social) and `IntentCRUD` (for tasks) instead of defaulting to the restrictive `IntentVibe`.
- [ ] **Telegram Authorization Feedback**: Send an explicit "Access Denied" or "Pending Authorization" message instead of silently dropping unauthorized chats.

## Long-term Stability
- [ ] **Daemon Self-Healing**: Implement logic in `Butler.WatchHealth` to automatically restart the `vibeaura` daemon if the socket is unresponsive.
- [ ] **Connection Pooling**: Reuse UDS connections where possible instead of dialing a new socket for every query.
