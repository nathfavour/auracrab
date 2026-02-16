# Browser Automation

Auracrab features a unique "Zero-Headless" browser automation system that operates directly within your active browser session.

## Why Zero-Headless?

Traditional automation uses headless browsers (like Puppeteer or Playwright in a separate window). Auracrab uses a **Browser Extension Bridge**, which means:
- **Existing Auth**: Use your already logged-in sessions (Twitter, Gmail, GitHub, etc.).
- **Bot Detection**: Harder to detect as it runs in a real browser with real user agents.
- **Visual Feedback**: You can see exactly what the agent is doing in your current tab.

## Core Actions

The `browser` skill supports the following atomic actions:
- `open`: Opens a URL or performs a search.
- `scrape`: Extracts text content from the page.
- `scrape_interactive`: Returns a JSON list of all buttons, links, and inputs.
- `click`: Performs a human-like click with smooth scrolling.
- `type`: Simulates human typing with varied delays.
- `hover`: Moves the virtual cursor over an element.
- `wait`: Waits for a selector to appear or a specific duration.
- `screenshot`: Captures the visible portion of the tab.

## Autonomous Agent (`browser_agent`)

The `browser_agent` is an autonomous loop that uses the atomic actions above to achieve complex goals.

### How it works:
1. **Observe**: Scrapes the interactive elements of the current page.
2. **Plan**: Asks the LLM for the next logical step based on the goal and history.
3. **Execute**: Performs the action (click, type, open, etc.).
4. **Repeat**: Continues until the goal is met or the max steps are reached.

### Example Usage:
```json
{
  "skill": "browser_agent",
  "args": {
    "goal": "Find the price of ETH on CoinMarketCap and tell me",
    "context": "coinmarketcap"
  }
}
```

## Context Awareness

Auracrab tracks all open tabs in your browser. You can target specific tasks to specific tabs using the `context` parameter (e.g., `context: "twitter"`). If a matching tab isn't found, the agent can use its "Intelligent URL Discovery" to find the right site.
