package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/helgesverre/dbdump/internal/database"
)

// TableSelectionModel represents the interactive table selection UI
type TableSelectionModel struct {
	tables   []database.TableInfo
	selected map[string]bool
	cursor   int
	done     bool
}

// NewTableSelectionModel creates a new table selection model
func NewTableSelectionModel(tables []database.TableInfo, preSelected []string) TableSelectionModel {
	selected := make(map[string]bool)
	for _, table := range preSelected {
		selected[table] = true
	}

	return TableSelectionModel{
		tables:   tables,
		selected: selected,
		cursor:   0,
		done:     false,
	}
}

// Init initializes the model
func (m TableSelectionModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m TableSelectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.done = true
			return m, tea.Quit

		case "enter":
			m.done = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.tables)-1 {
				m.cursor++
			}

		case " ":
			// Toggle selection
			table := m.tables[m.cursor].Name
			m.selected[table] = !m.selected[table]
		}
	}

	return m, nil
}

// View renders the UI
func (m TableSelectionModel) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("  Select tables to EXCLUDE data from (structure will be preserved)\n")
	b.WriteString("  Use ↑/↓ or j/k to move, SPACE to toggle, ENTER to confirm\n\n")

	for i, table := range m.tables {
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}

		checkbox := "☐"
		if m.selected[table.Name] {
			checkbox = "☑"
		}

		line := fmt.Sprintf("  %s %s %-30s (%s, %d rows)\n",
			cursor,
			checkbox,
			table.Name,
			table.SizeDisplay,
			table.RowCount,
		)

		b.WriteString(line)
	}

	b.WriteString("\n")

	return b.String()
}

// GetSelected returns the list of selected table names
func (m TableSelectionModel) GetSelected() []string {
	var selected []string
	for table, isSelected := range m.selected {
		if isSelected {
			selected = append(selected, table)
		}
	}
	return selected
}

// RunInteractiveSelection runs the interactive table selection
func RunInteractiveSelection(tables []database.TableInfo, preSelected []string) ([]string, error) {
	model := NewTableSelectionModel(tables, preSelected)

	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run interactive selection: %w", err)
	}

	m := finalModel.(TableSelectionModel)
	return m.GetSelected(), nil
}
