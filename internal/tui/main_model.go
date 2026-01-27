package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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
	"github.com/nathfavour/auracrab/pkg/core"
	"github.com/nathfavour/auracrab/pkg/skills"
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
