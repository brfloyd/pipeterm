package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type (
	errMsg           error
	TextInputDoneMsg string
)
type TextInputModel struct {
	textInput textinput.Model
	err       error
	width     int
	height    int
}

func newTextInput() *TextInputModel {
	ti := textinput.New()
	ti.Placeholder = "type name..."
	ti.Focus()
	ti.CharLimit = 20
	ti.Width = 20

	return &TextInputModel{
		textInput: ti,
		err:       nil,
	}
}

func (namePipeline *TextInputModel) Init() tea.Cmd {
	return textinput.Blink
}
func (namePipeline *TextInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			return namePipeline, func() tea.Msg {
				return TextInputDoneMsg(namePipeline.textInput.Value())
			}
		case tea.KeyCtrlC, tea.KeyEsc:
			return namePipeline, tea.Quit
		}
	case errMsg:
		namePipeline.err = msg
		return namePipeline, nil
	}

	namePipeline.textInput, cmd = namePipeline.textInput.Update(msg)
	return namePipeline, cmd
}

func (namePipeline *TextInputModel) View() string {

	return namePipeline.textInput.View()

}
