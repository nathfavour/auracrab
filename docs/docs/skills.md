# Skill Reference

Skills are the capabilities Auracrab agents use to interact with the world.

## `browser`
Direct control over the browser session.
- **Actions**: `open`, `scrape`, `scrape_interactive`, `click`, `type`, `hover`, `wait`, `screenshot`.
- **Key Params**: `url`, `selector`, `text`, `condition`, `context`.

## `browser_agent`
Autonomous multi-step browser automation.
- **Params**:
    - `goal` (required): The objective to achieve.
    - `context`: Filter for specific browser profiles/tabs.
    - `max_steps`: Maximum number of iterations (default: 10).

## `vault`
Secure credential management.
- **Actions**: `set`, `get`, `list`, `delete`.
- **Usage**: Used to store API tokens and secrets used by other skills.

## `social`
Integration with communication platforms.
- **Platforms**: Telegram, Discord.
- **Capabilities**: Send and receive messages, monitor channels.

## `autocommit`
Automated git workflow management.
- **Actions**: Analyzes changes and generates semantic commit messages.

## `system`
Local environment interaction.
- **Actions**: `exec` (runs shell commands), `read_file`, `write_file`.
- **Security**: Commands are typically executed in a sandboxed environment.
