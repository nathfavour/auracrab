# Getting Started

Get Auracrab up and running on your system in minutes.

## Installation

### Prerequisites
- [Go](https://go.dev/doc/install) 1.21 or later.
- [Node.js](https://nodejs.org/) and [pnpm](https://pnpm.io/) (for the browser extension).

### Quick Install (Shell)
```bash
curl -sSL https://raw.githubusercontent.com/nathfavour/auracrab/main/install.sh | bash
```

### Build from Source
1. Clone the repository:
   ```bash
   git clone https://github.com/nathfavour/auracrab.git
   cd auracrab
   ```
2. Build the binary:
   ```bash
   make build
   ```
3. Install the browser extension:
   ```bash
   cd webuse
   pnpm install
   pnpm dev # This will open a browser with the extension loaded
   ```

## Initial Configuration

1. **Start the Daemon**:
   ```bash
   auracrab start
   ```
2. **Setup the Vault**:
   Store your API keys (e.g., Gemini, Telegram, Discord) securely:
   ```bash
   auracrab vault set GEMINI_API_KEY your_key_here
   ```
3. **Connect the Browser**:
   Install and open the Auracrab extension in your browser. It should automatically connect to the daemon on port `9999`.

## Your First Command

Open the interactive TUI to monitor your butler:
```bash
auracrab vibe
```

Or run a simple browser task:
```bash
auracrab browser open https://google.com
```
