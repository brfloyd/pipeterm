package main

import (
	"fmt"
	"github.com/brfloyd/senior-project-brett-cli-data-project/tui"
	tea "github.com/charmbracelet/bubbletea"
	"os"
)

func main() {

	model := tui.InitialModel()
	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
