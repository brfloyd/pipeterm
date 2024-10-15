package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type QueryEditor struct {
	textarea textarea.Model
	viewport viewport.Model
	dataLake string
	width    int
	height   int
}

func NewQueryEditor(dataLake string, width, height int) *QueryEditor {
	ta := textarea.New()
	ta.Placeholder = "Enter your SQL query..."
	ta.Focus()
	ta.CharLimit = 0 // No character limit
	ta.SetWidth(width)
	ta.SetHeight(height - 5)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = true
	ta.KeyMap.InsertNewline.SetEnabled(true)
	vp := viewport.New(30, 5)
	vp.SetContent("Query results will appear here.")

	return &QueryEditor{
		textarea: ta,
		viewport: vp,
		dataLake: dataLake,
		width:    width,
		height:   height,
	}
}

func (qe *QueryEditor) Update(msg tea.Msg) (*QueryEditor, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		qe.width = msg.Width
		qe.height = msg.Height
		qe.viewport.Width = msg.Width
		qe.viewport.Height = msg.Height - 5
		qe.textarea.SetWidth(msg.Width)
		return qe, nil

	case tea.KeyMsg:
		switch {
		case msg.String() == "esc":
			// Exit the query editor
			return qe, func() tea.Msg { return exitEditorMsg{} }

		case msg.Type == tea.KeyCtrlE:
			// Execute the query
			query := qe.textarea.Value()
			return qe, executeQueryCmd(qe.dataLake, query)

		default:
			qe.textarea, cmd = qe.textarea.Update(msg)
			return qe, cmd
		}

	case queryResultMsg:
		if msg.err != nil {
			qe.viewport.SetContent(fmt.Sprintf("Error: %v", msg.err))
		} else {
			qe.viewport.SetContent(msg.result)
		}
		return qe, nil

	default:
		qe.textarea, cmd = qe.textarea.Update(msg)
		return qe, cmd
	}
}

func (qe *QueryEditor) View() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		qe.viewport.View(),
		qe.textarea.View(),
	)
}

type exitEditorMsg struct{}
