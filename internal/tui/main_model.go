package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathfavour/auracrab/pkg/core"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	taskStyle = lipgloss.NewStyle().
			PaddingLeft(2)
)

type tickMsg time.Time

type Model struct {
	tasks     []*core.Task
	cursor    int
	statusMsg string
}

func InitialModel() Model {
	return Model{
		tasks: core.GetButler().ListTasks(),
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
	case tickMsg:
		m.tasks = core.GetButler().ListTasks()
		m.statusMsg = core.GetButler().GetStatus()
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
	s := titleStyle.Render("🦀 AURACRAB  - TUI") + "\n\n"
	s += statusStyle.Render(m.statusMsg) + "\n\n"

	s += "Managed Tasks:\n"
	if len(m.tasks) == 0 {
		s += "  (no tasks yet)\n"
	}

	for i, task := range m.tasks {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		statusIcon := "⏳"
		if task.Status == core.TaskStatusCompleted {
			statusIcon = "✅"
		} else if task.Status == core.TaskStatusFailed {
			statusIcon = "❌"
		}

		s += fmt.Sprintf("%s %s %s - %s\n", cursor, statusIcon, task.ID, task.Content)
	}

	s += "\n[q]uit [r]efresh\n"

	return s
}
