package main

import (
    "fmt"
    "os"
    "github.com/brfloyd/senior-project-brett-cli-data-project/tui"  
    tea "github.com/charmbracelet/bubbletea"
)

func main() {
    
    model := tui.InitialModel()  
    p := tea.NewProgram(model)  
    if err := p.Start(); err != nil {
        fmt.Println("Error running program:", err)
        os.Exit(1)
    }
}

