package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/nathfavour/auracrab/pkg/config"
	"github.com/nathfavour/auracrab/pkg/core"
	"github.com/nathfavour/auracrab/pkg/skills"
	"github.com/nathfavour/auracrab/pkg/vault"
)

var (
	// Colors
	purple = lipgloss.Color("#7D56F4")
	green  = lipgloss.Color("#04B575")
	red    = lipgloss.Color("#ED567A")
	gray   = lipgloss.Color("#626262")
	white  = lipgloss.Color("#FAFAFA")

	// Styles
	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(white).
			Background(purple).
			Padding(0, 1).
			MarginBottom(1)

	styleSectionTitle = lipgloss.NewStyle().
				Bold(true).
				Foreground(purple).
				MarginTop(1).
				MarginBottom(1)

	styleTaskCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(gray).
			Padding(0, 1).
			MarginBottom(1).
			Width(60)

	styleSelectedTask = styleTaskCard.Copy().
				BorderForeground(purple).
				Background(lipgloss.Color("#1A1A1A"))

	styleSidebar = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(gray).
			PaddingLeft(2).
			MarginLeft(2).
			Width(30)

	styleFooter = lipgloss.NewStyle().
			Foreground(gray).
			MarginTop(1)

	styleHealthOk   = lipgloss.NewStyle().Foreground(green)
	styleHealthWarn = lipgloss.NewStyle().Foreground(red)

	styleInput = lipgloss.NewStyle().
			Foreground(white).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(purple).
			Padding(0, 1).
			Width(90)

	styleStatus = lipgloss.NewStyle().
			Italic(true).
			Foreground(gray).
			MarginLeft(2)
)

type tickMsg time.Time

type Model struct {
	tasks          []*core.Task
	cursor         int
	statusMsg      string
	healthMsg      string
	skillsList     []string
	width          int
	height         int
	ready          bool
	input          textinput.Model
	banner         string
	lastResponse   string
	isCapturing    bool
	updateStatus   string
	// History fields
	commandHistory []string
	historyIndex   int
	// Config mode fields
	isConfiguring  bool
	configSteps    []configStep
	currentStep    int
	configValues   map[string]string
	configuringFor string
}

type configStep struct {
	question    string
	vaultKey    string
	placeholder string
	sensitive   bool
}

func InitialModel() Model {
	butler := core.GetButler()

	ti := textinput.New()
	ti.Placeholder = "Enter task or /command..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 80

	var skillNames []string
	v := vault.GetVault()
	for _, s := range skills.GetRegistry().List() {
		enabled, _ := v.Get(strings.ToUpper(s.Name()) + "_ENABLED")
		if enabled == "" || enabled == "true" {
			skillNames = append(skillNames, s.Name())
		}
	}

	return Model{
		tasks:        butler.ListTasks(),
		statusMsg:    butler.GetStatus(),
		healthMsg:    butler.WatchHealth(),
		skillsList:   skillNames,
		input:        ti,
		configValues: make(map[string]string),
		historyIndex: -1,
	}
}

func (m Model) Init() tea.Cmd {
	return tick()
}

func tick() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.banner = buildBanner(m.width)

	case tickMsg:
		butler := core.GetButler()
		m.tasks = butler.ListTasks()
		m.statusMsg = butler.GetStatus()
		m.healthMsg = butler.WatchHealth()

		var skillNames []string
		v := vault.GetVault()
		for _, s := range skills.GetRegistry().List() {
			enabled, _ := v.Get(strings.ToUpper(s.Name()) + "_ENABLED")
			if enabled == "" || enabled == "true" {
				skillNames = append(skillNames, s.Name())
			}
		}
		m.skillsList = skillNames

		// Check for background updates
		availFile := filepath.Join(config.DataDir(), ".update_available")
		completeFile := filepath.Join(config.DataDir(), ".update_complete")

		if _, err := os.Stat(completeFile); err == nil {
			m.updateStatus = "✨ Update completed! Restart required."
		} else if _, err := os.ReadFile(availFile); err == nil {
			m.updateStatus = "✨ Update downloading in background..."
		} else {
			m.updateStatus = ""
		}

		return m, tick()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.isConfiguring {
				m.isConfiguring = false
				m.lastResponse = "Setup cancelled."
				m.input.EchoMode = textinput.EchoNormal
				m.input.Placeholder = "Enter task or /command..."
				m.input.SetValue("")
				return m, tea.Batch(cmds...)
			}
		case "up", "k":
			if m.isConfiguring {
				return m, nil
			}
			if len(m.commandHistory) > 0 {
				if m.historyIndex == -1 {
					m.historyIndex = len(m.commandHistory) - 1
				} else if m.historyIndex > 0 {
					m.historyIndex--
				}
				m.input.SetValue(m.commandHistory[m.historyIndex])
				m.input.CursorEnd()
				return m, nil
			}
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.isConfiguring {
				return m, nil
			}
			if m.historyIndex != -1 {
				if m.historyIndex < len(m.commandHistory)-1 {
					m.historyIndex++
					m.input.SetValue(m.commandHistory[m.historyIndex])
					m.input.CursorEnd()
				} else {
					m.historyIndex = -1
					m.input.SetValue("")
				}
				return m, nil
			}
			if m.cursor < len(m.tasks)-1 {
				m.cursor++
			}
		case "enter":
			if m.isConfiguring {
				// ... (config logic)
				val := m.input.Value()
				m.input.SetValue("")
				m.input.EchoMode = textinput.EchoNormal

				step := m.configSteps[m.currentStep]
				m.configValues[step.vaultKey] = val

				m.currentStep++
				if m.currentStep >= len(m.configSteps) {
					// Finish setup
					m.isConfiguring = false
					v := vault.GetVault()
					for k, vStr := range m.configValues {
						_ = v.Set(k, vStr)
					}
					// Always enable if setting up
					_ = v.Set(strings.ToUpper(m.configuringFor)+"_ENABLED", "true")
					
					m.configValues = make(map[string]string) // Clear sensitive values from memory
					m.lastResponse = fmt.Sprintf("✅ Setup for %s completed. Restart the daemon to apply changes.", m.configuringFor)
					m.input.Placeholder = "Enter task or /command..."
				} else {
					// Next step
					nextStep := m.configSteps[m.currentStep]
					m.input.Placeholder = nextStep.placeholder
					if nextStep.sensitive {
						m.input.EchoMode = textinput.EchoPassword
					}
					m.lastResponse = nextStep.question
				}
				return m, tea.Batch(cmds...)
			}

			if m.input.Value() != "" {
				val := m.input.Value()
				m.input.SetValue("")
				m.historyIndex = -1

				// Append to history if not same as last
				if len(m.commandHistory) == 0 || m.commandHistory[len(m.commandHistory)-1] != val {
					m.commandHistory = append(m.commandHistory, val)
				}

				if strings.HasPrefix(val, "/") {
					return m.handleCommand(val)
				}

				// Proactive check for module setup
				v := vault.GetVault()
				lowerVal := strings.ToLower(val)
				if strings.Contains(lowerVal, "telegram") && (strings.Contains(lowerVal, "init") || strings.Contains(lowerVal, "setup") || strings.Contains(lowerVal, "configure") || strings.Contains(lowerVal, "start")) {
					token, _ := v.Get("TELEGRAM_TOKEN")
					if token == "" {
						return m.handleSetupCommand("telegram")
					}
				} else if strings.Contains(lowerVal, "discord") && (strings.Contains(lowerVal, "init") || strings.Contains(lowerVal, "setup") || strings.Contains(lowerVal, "configure") || strings.Contains(lowerVal, "start")) {
					token, _ := v.Get("DISCORD_TOKEN")
					if token == "" {
						return m.handleSetupCommand("discord")
					}
				}

				// Start as task
				_, _ = core.GetButler().StartTask(context.Background(), val, "")
				m.lastResponse = "Task started: " + val
			}
		}
	}

	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) handleCommand(input string) (tea.Model, tea.Cmd) {
	parts := strings.Split(input, " ")
	cmd := parts[0]

	switch cmd {
	case "/config":
		if len(parts) < 2 {
			m.lastResponse = "Usage: /config [list|set <key> <val>|get <key>|toggle <key>]"
			return m, nil
		}
		return m.handleConfigCommand(parts[1:])
	case "/setup":
		if len(parts) < 2 {
			m.lastResponse = "Usage: /setup [telegram|discord]"
			return m, nil
		}
		return m.handleSetupCommand(parts[1])
	case "/shot":
		return m.takeScreenshot()
	case "/exit", "/quit":
		return m, tea.Quit
	case "/help":
		m.lastResponse = "Commands: /shot - Take screenshot, /config - Manage settings, /exit - Quit"
	default:
		m.lastResponse = "Unknown command: " + cmd
	}
	return m, nil
}

func (m Model) handleSetupCommand(module string) (tea.Model, tea.Cmd) {
	m.configuringFor = strings.ToLower(module)
	m.isConfiguring = true
	m.currentStep = 0
	m.configValues = make(map[string]string)

	switch m.configuringFor {
	case "telegram":
		m.configSteps = []configStep{
			{
				question:    "Enter your Telegram Bot Token:",
				vaultKey:    "TELEGRAM_TOKEN",
				placeholder: "123456:ABC-DEF...",
				sensitive:   true,
			},
			{
				question:    "Enter Allowed Chat IDs (comma-separated, optional):",
				vaultKey:    "TELEGRAM_ALLOWED_CHATS",
				placeholder: "12345678,98765432",
				sensitive:   false,
			},
		}
	case "discord":
		m.configSteps = []configStep{
			{
				question:    "Enter your Discord Bot Token:",
				vaultKey:    "DISCORD_TOKEN",
				placeholder: "OTY...",
				sensitive:   true,
			},
		}
	default:
		m.isConfiguring = false
		m.lastResponse = "Unknown module for setup: " + module
		return m, nil
	}

	firstStep := m.configSteps[0]
	m.lastResponse = "🔒 Configuration Mode - " + firstStep.question
	m.input.Placeholder = firstStep.placeholder
	if firstStep.sensitive {
		m.input.EchoMode = textinput.EchoPassword
	} else {
		m.input.EchoMode = textinput.EchoNormal
	}
	m.input.SetValue("")

	return m, nil
}

func (m Model) handleConfigCommand(args []string) (tea.Model, tea.Cmd) {
	v := vault.GetVault()
	sub := args[0]

	switch sub {
	case "list":
		keys, err := v.List()
		if err != nil {
			m.lastResponse = "Error listing configs: " + err.Error()
		} else if len(keys) == 0 {
			m.lastResponse = "No configurations found."
		} else {
			m.lastResponse = "Configs: " + strings.Join(keys, ", ")
		}
	case "set":
		if len(args) < 3 {
			m.lastResponse = "Usage: /config set <key> <value>"
		} else {
			err := v.Set(args[1], args[2])
			if err != nil {
				m.lastResponse = "Error setting config: " + err.Error()
			} else {
				m.lastResponse = fmt.Sprintf("Config '%s' set to '%s'", args[1], args[2])
			}
		}
	case "get":
		if len(args) < 2 {
			m.lastResponse = "Usage: /config get <key>"
		} else {
			val, err := v.Get(args[1])
			if err != nil {
				m.lastResponse = "Error getting config: " + err.Error()
			} else {
				m.lastResponse = fmt.Sprintf("%s: %s", args[1], val)
			}
		}
	case "toggle":
		if len(args) < 2 {
			m.lastResponse = "Usage: /config toggle <key>"
		} else {
			key := args[1]
			val, err := v.Get(key)
			newVal := "true"
			if err == nil && (val == "true" || val == "on" || val == "yes" || val == "1") {
				newVal = "false"
			}
			err = v.Set(key, newVal)
			if err != nil {
				m.lastResponse = "Error toggling config: " + err.Error()
			} else {
				m.lastResponse = fmt.Sprintf("Config '%s' toggled to %s", key, newVal)
			}
		}
	default:
		m.lastResponse = "Unknown config subcommand: " + sub
	}

	return m, nil
}

func (m Model) View() string {
	if !m.ready {
		return "Initializing Auracrab TUI..."
	}

	var view strings.Builder

	// Banner
	view.WriteString(m.banner + "\n\n")

	// Left Column: Tasks
	var taskList strings.Builder
	taskList.WriteString(styleSectionTitle.Render("DELEGATED TASKS") + "\n")

	if len(m.tasks) == 0 {
		taskList.WriteString(styleFooter.Render("  No active tasks. Enter a task below to start."))
	} else {
		for i, task := range m.tasks {
			statusIcon := "⏳"
			if task.Status == core.TaskStatusCompleted {
				statusIcon = "✅"
			} else if task.Status == core.TaskStatusFailed {
				statusIcon = "❌"
			}

			content := task.Content
			if len(content) > 50 {
				content = content[:47] + "..."
			}

			cardContent := fmt.Sprintf("%s %s\n%s", statusIcon, task.ID, content)

			if m.cursor == i {
				taskList.WriteString(styleSelectedTask.Render(cardContent) + "\n")
			} else {
				taskList.WriteString(styleTaskCard.Render(cardContent) + "\n")
			}
		}
	}

	// Right Column: System Info
	var sidebar strings.Builder
	sidebar.WriteString(styleSectionTitle.Render("SYSTEM VIBES") + "\n")

	healthStyle := styleHealthOk
	if strings.Contains(strings.ToLower(m.healthMsg), "warning") || strings.Contains(strings.ToLower(m.healthMsg), "anomaly") {
		healthStyle = styleHealthWarn
	}

	sidebar.WriteString("Health: " + healthStyle.Render(m.healthMsg) + "\n\n")
	sidebar.WriteString("Status: " + m.statusMsg + "\n\n")

	if m.updateStatus != "" {
		sidebar.WriteString(lipgloss.NewStyle().Foreground(green).Bold(true).Render(m.updateStatus) + "\n\n")
	}

	sidebar.WriteString(styleSectionTitle.Render("LOADED SKILLS") + "\n")
	for _, s := range m.skillsList {
		sidebar.WriteString("• " + s + "\n")
	}

	// Layout Main
	mainContent := lipgloss.JoinHorizontal(lipgloss.Top,
		taskList.String(),
		styleSidebar.Render(sidebar.String()),
	)
	view.WriteString(mainContent)

	// Footer: Input & Status
	view.WriteString("\n\n")
	if m.lastResponse != "" {
		if m.isConfiguring {
			view.WriteString(lipgloss.NewStyle().Foreground(purple).Bold(true).Render(" 🔒 CONFIG MODE: ") + m.lastResponse + "\n")
		} else {
			view.WriteString(styleStatus.Render(m.lastResponse) + "\n")
		}
	}
	view.WriteString(styleInput.Render(m.input.View()))

	if m.isConfiguring {
		view.WriteString(styleFooter.Render("\n[Enter] Confirm Step • [Ctrl+C] Cancel Setup"))
	} else {
		view.WriteString(styleFooter.Render("\n[↑/↓] Navigate • [Enter] Submit • [/shot] Screenshot • [/config] Config • [/setup] Setup • [Ctrl+C] Quit"))
	}

	return view.String()
}

func buildBanner(width int) string {
	if width <= 0 {
		width = 60
	}

	ascii := []string{
		lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true).Render(`      _   _ _ ____      _    ____ ____      _    ____  `),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#8700FF")).Bold(true).Render(`     / \ | | |  _ \    / \  / ___|  _ \    / \  | __ ) `),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#AF00FF")).Bold(true).Render(`    / _ \| | | |_) |  / _ \| |   | |_) |  / _ \|  _ \ `),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#D700FF")).Bold(true).Render(`   / ___ \ |_| |  _ <  / ___ \ |___|  _ <  / ___ \ |_) |`),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#FF00D7")).Bold(true).Render(`  /_/   \_\___/|_| \_\/_/   \_\____|_| \_\/_/   \_\____/ `),
	}

	maxASCII := 0
	for _, l := range ascii {
		w := lipgloss.Width(l)
		if w > maxASCII {
			maxASCII = w
		}
	}
	if maxASCII < 1 {
		maxASCII = 1
	}

	tagline := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render("Autonomous, Persistent AI Agent Daemon for the Agentic Era")

	if width >= maxASCII {
		return strings.Join(append(ascii, "\n"+tagline), "\n")
	}

	// Compact
	return gradientWord("AURACRAB", []lipgloss.Color{
		lipgloss.Color("#FF00D7"),
		lipgloss.Color("#D700FF"),
		lipgloss.Color("#AF00FF"),
		lipgloss.Color("#8700FF"),
		lipgloss.Color("#7D56F4"),
	}, true) + "\n" + tagline
}

func gradientWord(word string, colors []lipgloss.Color, spaced bool) string {
	var sb strings.Builder
	for i, r := range word {
		color := colors[i%len(colors)]
		char := string(r)
		if spaced && i < len(word)-1 {
			char += " "
		}
		sb.WriteString(lipgloss.NewStyle().Foreground(color).Bold(true).Render(char))
	}
	return sb.String()
}

func (m Model) takeScreenshot() (tea.Model, tea.Cmd) {
	dir := config.ScreenshotDir()

	timestamp := time.Now().Format("2006-01-02_150405")
	filename := fmt.Sprintf("auracrab_%s", timestamp)

	basePath := filepath.Join(dir, filename)
	svgPath := basePath + ".svg"
	pngPath := basePath + ".png"

	m.isCapturing = true
	rawView := m.View()
	m.isCapturing = false

	svgContent := convertAnsiToSVG(rawView)
	_ = os.WriteFile(svgPath, []byte(svgContent), 0644)

	err := m.convertToPNG(svgPath, pngPath)
	if err != nil {
		m.lastResponse = "Captured SVG (PNG failed: " + err.Error() + ")"
	} else {
		m.lastResponse = "Screenshot saved: " + pngPath
	}

	return m, nil
}

// --- Screenshot Helpers ---

type ansiPart struct {
	text string
	fg   string
	bold bool
}

// convertAnsiToSVG converts colored terminal output to a styled SVG ensemble
func convertAnsiToSVG(ansi string) string {
	lines := strings.Split(ansi, "\n")

	// Keep only SGR sequences (colors/styles). Remove cursor/alt-screen/etc.
	reSGR := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	reCSI := regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)
	reOSC := regexp.MustCompile(`\x1b\][^\x07]*(\x07|\x1b\\)`)

	cleanLines := make([]string, 0, len(lines))
	for _, l := range lines {
		cleanLines = append(cleanLines, sanitizeANSI(l, reCSI, reOSC))
	}

	// Detect a common full-width right border column (Lipgloss borders often
	// render a vertical bar at the terminal width, making screenshots massive).
	borderCol := detectRightBorderColumn(cleanLines, reSGR)

	// Compute real content width (in terminal columns, not bytes), trimming
	// trailing whitespace and ignoring the detected right-side border.
	maxCols := 0
	for _, l := range cleanLines {
		cols := visibleTrimmedWidth(l, reSGR)
		if borderCol > 0 && cols == borderCol {
			if r, ok := lastNonSpaceRune(reSGR.ReplaceAllString(l, "")); ok && isBorderRune(r) {
				cols -= runewidth.RuneWidth(r)
			}
		}
		if cols > maxCols {
			maxCols = cols
		}
	}
	if maxCols < 1 {
		maxCols = 1
	}

	// Truncate lines to the computed width so the rendered SVG is actually cropped.
	for i := range cleanLines {
		cleanLines[i] = truncateAnsiLineToWidth(cleanLines[i], maxCols, reSGR)
	}

	// Refined dimensions
	fontSize := 14
	lineHeight := 1.25
	charWidth := 8.2

	paddingX := 30.0
	paddingY := 60.0

	width := float64(maxCols)*charWidth + (paddingX * 2)
	height := float64(len(cleanLines))*float64(fontSize)*lineHeight + paddingY + 40

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg width="%.1f" height="%.1f" viewBox="0 0 %.1f %.1f" xmlns="http://www.w3.org/2000/svg">`, width, height, width, height))

	// Add Shadow
	sb.WriteString(fmt.Sprintf(`<rect x="15" y="15" width="%.1f" height="%.1f" rx="12" fill="rgba(0,0,0,0.4)" filter="blur(8px)" />`, width-20, height-20))

	// Main Frame
	sb.WriteString(fmt.Sprintf(`<rect x="10" y="10" width="%.1f" height="%.1f" rx="12" fill="#0D0D0D" stroke="#7D56F4" stroke-width="2" />`, width-20, height-20))

	// Title/Controls dots (Mac style)
	sb.WriteString(`<circle cx="35" cy="30" r="5" fill="#FF5F56"/>`)
	sb.WriteString(`<circle cx="55" cy="30" r="5" fill="#FFBD2E"/>`)
	sb.WriteString(`<circle cx="75" cy="30" r="5" fill="#27C93F"/>`)

	sb.WriteString(`<text font-family="Menlo, Monaco, Consolas, Courier New, monospace" font-size="14" xml:space="preserve">`)

	for i, line := range cleanLines {
		yPos := 70 + (i * int(float64(fontSize)*lineHeight))
		sb.WriteString(fmt.Sprintf(`<tspan x="%d" y="%d">`, int(paddingX), yPos))

		parts := parseAnsiLine(line, reSGR)
		for _, p := range parts {
			style := ""
			if p.fg != "" {
				style += fmt.Sprintf("fill:%s;", p.fg)
			} else {
				style += "fill:#FAFAFA;"
			}
			if p.bold {
				style += "font-weight:bold;"
			}

			escapedText := strings.ReplaceAll(p.text, "&", "&amp;")
			escapedText = strings.ReplaceAll(escapedText, "<", "&lt;")
			escapedText = strings.ReplaceAll(escapedText, ">", "&gt;")
			// Ensure spaces are visible
			escapedText = strings.ReplaceAll(escapedText, " ", "&#160;")

			sb.WriteString(fmt.Sprintf(`<tspan style="%s">%s</tspan>`, style, escapedText))
		}
		sb.WriteString(`</tspan>`)
	}

	sb.WriteString(`</text></svg>`)
	return sb.String()
}

func sanitizeANSI(line string, reCSI, reOSC *regexp.Regexp) string {
	// Strip OSC sequences entirely (titles, hyperlinks, etc).
	line = reOSC.ReplaceAllString(line, "")
	// Strip CSI sequences unless they are SGR (ending with 'm').
	return reCSI.ReplaceAllStringFunc(line, func(seq string) string {
		if strings.HasSuffix(seq, "m") {
			return seq
		}
		return ""
	})
}

func visibleTrimmedWidth(line string, reSGR *regexp.Regexp) int {
	visible := reSGR.ReplaceAllString(line, "")
	visible = strings.TrimRight(visible, " \t")
	return runewidth.StringWidth(visible)
}

func lastNonSpaceRune(s string) (rune, bool) {
	s = strings.TrimRight(s, " \t")
	if s == "" {
		return 0, false
	}
	r, _ := utf8.DecodeLastRuneInString(s)
	return r, true
}

func isBorderRune(r rune) bool {
	switch r {
	case '|',
		'│', '┃', '║',
		'┤', '├', '┐', '┘', '┌', '└',
		'┬', '┴', '┼',
		'╡', '╢', '╣', '╠', '╞',
		'╭', '╮', '╯', '╰',
		'─', '━', '═':
		return true
	default:
		return false
	}
}

func detectRightBorderColumn(lines []string, reSGR *regexp.Regexp) int {
	counts := map[int]int{}
	for _, l := range lines {
		visible := reSGR.ReplaceAllString(l, "")
		visible = strings.TrimRight(visible, " \t")
		if visible == "" {
			continue
		}
		last, ok := lastNonSpaceRune(visible)
		if !ok || !isBorderRune(last) {
			continue
		}
		col := runewidth.StringWidth(visible)
		counts[col]++
	}

	bestCol := 0
	bestCount := 0
	for col, count := range counts {
		if count > bestCount {
			bestCol = col
			bestCount = count
		}
	}

	// Heuristic: if many lines share the same ending border column, treat it as a
	// full-width frame and crop it away.
	if bestCount >= 3 && bestCount >= len(lines)/3 {
		return bestCol
	}
	return 0
}

func truncateAnsiLineToWidth(line string, maxCols int, reSGR *regexp.Regexp) string {
	if maxCols <= 0 || line == "" {
		return ""
	}

	indices := reSGR.FindAllStringIndex(line, -1)
	var b strings.Builder
	visibleCols := 0
	lastEnd := 0

	writeText := func(segment string) bool {
		for _, r := range segment {
			rw := runewidth.RuneWidth(r)
			if rw == 0 {
				rw = 1
			}
			if visibleCols+rw > maxCols {
				return false
			}
			b.WriteRune(r)
			visibleCols += rw
		}
		return true
	}

	for _, idx := range indices {
		if idx[0] > lastEnd {
			if !writeText(line[lastEnd:idx[0]]) {
				return b.String()
			}
		}
		if visibleCols >= maxCols {
			return b.String()
		}
		b.WriteString(line[idx[0]:idx[1]])
		lastEnd = idx[1]
	}

	if lastEnd < len(line) {
		_ = writeText(line[lastEnd:])
	}
	return b.String()
}

func parseAnsiLine(line string, re *regexp.Regexp) []ansiPart {
	var parts []ansiPart
	currFg := "#FAFAFA"
	currBold := false

	indices := re.FindAllStringIndex(line, -1)
	lastEnd := 0

	for _, idx := range indices {
		if idx[0] > lastEnd {
			parts = append(parts, ansiPart{text: line[lastEnd:idx[0]], fg: currFg, bold: currBold})
		}

		code := line[idx[0]:idx[1]]
		if code == "\x1b[0m" {
			currFg = "#FAFAFA"
			currBold = false
		} else {
			// Handle TrueColor: \x1b[38;2;r;g;bm
			if strings.Contains(code, "38;2;") {
				clean := strings.Trim(code, "\x1b[m")
				pts := strings.Split(clean, ";")
				if len(pts) >= 5 {
					r, _ := strconv.Atoi(pts[2])
					g, _ := strconv.Atoi(pts[3])
					b, _ := strconv.Atoi(pts[4])
					currFg = fmt.Sprintf("#%02x%02x%02x", r, g, b)
				}
			} else if strings.Contains(code, "38;5;") {
				currFg = "#7D56F4"
			} else {
				// Map basic colors only if not TrueColor
				if strings.Contains(code, "35") {
					currFg = "#EE6FF8"
				} else if strings.Contains(code, "36") {
					currFg = "#04D9FF"
				} else if strings.Contains(code, "34") {
					currFg = "#7D56F4"
				}
			}

			if strings.Contains(code, ";1m") || strings.Contains(code, "[1;") || code == "\x1b[1m" {
				currBold = true
			}
		}
		lastEnd = idx[1]
	}

	if lastEnd < len(line) {
		parts = append(parts, ansiPart{text: line[lastEnd:], fg: currFg, bold: currBold})
	}

	return parts
}

// convertToPNG attempts to convert SVG to PNG using system tools
func (m Model) convertToPNG(svgPath, pngPath string) error {
	// Try rsvg-convert (common on Linux)
	if _, err := exec.LookPath("rsvg-convert"); err == nil {
		return exec.Command("rsvg-convert", "-o", pngPath, svgPath).Run()
	}

	// Try ImageMagick
	if _, err := exec.LookPath("magick"); err == nil {
		return exec.Command("magick", svgPath, pngPath).Run()
	} else if _, err := exec.LookPath("convert"); err == nil {
		return exec.Command("convert", svgPath, pngPath).Run()
	}

	// Try ffmpeg (common on Termux)
	if _, err := exec.LookPath("ffmpeg"); err == nil {
		return exec.Command("ffmpeg", "-i", svgPath, pngPath).Run()
	}

	return fmt.Errorf("no conversion tool found (rsvg-convert, magick, or ffmpeg)")
}
