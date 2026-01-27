package tui

import (
"fmt"
"strings"
"time"

tea "github.com/charmbracelet/bubbletea"
"github.com/charmbracelet/lipgloss"
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
)

type tickMsg time.Time

type Model struct {
	tasks      []*core.Task
	cursor     int
	statusMsg  string
	healthMsg  string
	skillsList []string
	width      int
	height     int
	ready      bool
}

func InitialModel() Model {
	butler := core.GetButler()
	registry := skills.GetRegistry()
	
	var skillNames []string
	// We can't easily list from registry without a new method, but we know the defaults
skillNames = []string{"browser", "social", "autocommit", "system"}

return Model{
tasks:      butler.ListTasks(),
statusMsg:  butler.GetStatus(),
healthMsg:  butler.WatchHealth(),
skillsList: skillNames,
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
switch msg := msg.(type) {
case tea.WindowSizeMsg:
m.width = msg.Width
m.height = msg.Height
m.ready = true

case tickMsg:
butler := core.GetButler()
m.tasks = butler.ListTasks()
m.statusMsg = butler.GetStatus()
m.healthMsg = butler.WatchHealth()
return m, tick()

case tea.KeyMsg:
switch msg.String() {
case "ctrl+c", "q":
return m, tea.Quit
case "up", "k":
if m.cursor > 0 {
m.cursor--
}
case "down", "j":
if m.cursor < len(m.tasks)-1 {
m.cursor++
}
case "r":
m.tasks = core.GetButler().ListTasks()
}
}
return m, nil
}

func (m Model) View() string {
if !m.ready {
return "Initializing Auracrab TUI..."
}

header := styleHeader.Render("🦀 AURACRAB  v1.0.0 - AGENTIC DAEMON")

// Left Column: Tasks
var taskList strings.Builder
taskList.WriteString(styleSectionTitle.Render("DELEGATED TASKS") + "\n")

if len(m.tasks) == 0 {
taskList.WriteString(styleFooter.Render("  No active tasks. Send a message via Telegram/Discord to start."))
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

// Layout
mainContent := lipgloss.JoinHorizontal(lipgloss.Top, 
taskList.String(),
styleSidebar.Render(sidebar.String()),
)

// Footer
footer := styleFooter.Render("\n[↑/↓] Navigate • [r] Refresh • [q] Quit • [v] View Output")

return lipgloss.JoinVertical(lipgloss.Left, header, mainContent, footer)
}
