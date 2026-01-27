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

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/nathfavour/auracrab/pkg/core"
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
	tasks        []*core.Task
	cursor       int
	statusMsg    string
	healthMsg    string
	skillsList   []string
	width        int
	height       int
	ready        bool
	input        textinput.Model
	banner       string
	lastResponse string
	isCapturing  bool
}

func InitialModel() Model {
	butler := core.GetButler()

	ti := textinput.New()
	ti.Placeholder = "Enter task or /command..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 80

	var skillNames []string
	skillNames = []string{"browser", "social", "autocommit", "system"}

	return Model{
		tasks:      butler.ListTasks(),
		statusMsg:  butler.GetStatus(),
		healthMsg:  butler.WatchHealth(),
		skillsList: skillNames,
		input:      ti,
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
		return m, tick()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.tasks)-1 {
				m.cursor++
			}
		case "enter":
			if m.input.Value() != "" {
				val := m.input.Value()
				m.input.SetValue("")

				if strings.HasPrefix(val, "/") {
					return m.handleCommand(val)
				}

				// Start as task
				_, _ = core.GetButler().StartTask(context.Background(), val)
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
	case "/shot":
		return m.takeScreenshot()
	case "/exit", "/quit":
		return m, tea.Quit
	case "/help":
		m.lastResponse = "Commands: /shot - Take screenshot, /exit - Quit"
	default:
		m.lastResponse = "Unknown command: " + cmd
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
		view.WriteString(styleStatus.Render(m.lastResponse) + "\n")
	}
	view.WriteString(styleInput.Render(m.input.View()))
	view.WriteString(styleFooter.Render("\n[↑/↓] Navigate • [Enter] Submit • [/shot] Screenshot • [Ctrl+C] Quit"))

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
	dir := "screenshots"
	if err := os.MkdirAll(dir, 0755); err != nil {
		m.lastResponse = "Screenshot Error: " + err.Error()
		return m, nil
	}

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

	err := convertToPNG(svgPath, pngPath)
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

func convertAnsiToSVG(ansi string) string {
	lines := strings.Split(ansi, "\n")
	reSGR := regexp.MustCompile(`\x1b\[[0-9;]*m`)

	maxCols := 0
	for _, l := range lines {
		visible := reSGR.ReplaceAllString(l, "")
		cols := runewidth.StringWidth(visible)
		if cols > maxCols {
			maxCols = cols
		}
	}
	if maxCols < 1 {
		maxCols = 1
	}

	fontSize := 14
	lineHeight := 1.25
	charWidth := 8.2
	paddingX := 30.0
	paddingY := 60.0

	width := float64(maxCols)*charWidth + (paddingX * 2)
	height := float64(len(lines))*float64(fontSize)*lineHeight + paddingY + 40

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg width="%.1f" height="%.1f" viewBox="0 0 %.1f %.1f" xmlns="http://www.w3.org/2000/svg">`, width, height, width, height))
	sb.WriteString(fmt.Sprintf(`<rect x="10" y="10" width="%.1f" height="%.1f" rx="12" fill="#0D0D0D" stroke="#7D56F4" stroke-width="2" />`, width-20, height-20))
	sb.WriteString(`<text font-family="monospace" font-size="14" xml:space="preserve">`)

	for i, line := range lines {
		yPos := 70 + (i * int(float64(fontSize)*lineHeight))
		sb.WriteString(fmt.Sprintf(`<tspan x="%d" y="%d">`, int(paddingX), yPos))

		parts := parseAnsiLine(line, reSGR)
		for _, p := range parts {
			style := "fill:" + p.fg + ";"
			if p.bold {
				style += "font-weight:bold;"
			}
			text := strings.ReplaceAll(strings.ReplaceAll(p.text, "&", "&amp;"), "<", "&lt;")
			text = strings.ReplaceAll(text, " ", "&#160;")
			sb.WriteString(fmt.Sprintf(`<tspan style="%s">%s</tspan>`, style, text))
		}
		sb.WriteString(`</tspan>`)
	}
	sb.WriteString(`</text></svg>`)
	return sb.String()
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
		} else if strings.Contains(code, "38;2;") {
			clean := strings.Trim(code, "\x1b[m")
			pts := strings.Split(clean, ";")
			if len(pts) >= 5 {
				r, _ := strconv.Atoi(pts[2])
				g, _ := strconv.Atoi(pts[3])
				b, _ := strconv.Atoi(pts[4])
				currFg = fmt.Sprintf("#%02x%02x%02x", r, g, b)
			}
		} else if strings.Contains(code, "1m") {
			currBold = true
		}
		lastEnd = idx[1]
	}
	if lastEnd < len(line) {
		parts = append(parts, ansiPart{text: line[lastEnd:], fg: currFg, bold: currBold})
	}
	return parts
}

func convertToPNG(svgPath, pngPath string) error {
	if _, err := exec.LookPath("rsvg-convert"); err == nil {
		return exec.Command("rsvg-convert", "-o", pngPath, svgPath).Run()
	}
	if _, err := exec.LookPath("magick"); err == nil {
		return exec.Command("magick", svgPath, pngPath).Run()
	}
	return fmt.Errorf("no conversion tool found")
}