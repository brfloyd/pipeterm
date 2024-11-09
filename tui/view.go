package tui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/common-nighthawk/go-figure"
)

func (m Model) View() string {
	s := ""

	// Quick Action Bar
	actionBarStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#333333")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Width(m.width).
		Align(lipgloss.Center).
		Padding(0, 1)

	actionBar := actionBarStyle.Render(" [c]Create Pipeline [?]Help [a]About [p]Pipelines [e]Query Editor [q]Quit ")
	s += actionBar + "\n\n"

	if m.inDataLakeSelect {
		unslectedLineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
		selectedLineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF7F00"))
		s += selectedLineStyle.Render("Select a Data Lake:\n")
		s += "\n"
		for i, lake := range m.dataLakes {
			cursor := "  "
			lineStyle := unslectedLineStyle
			if m.selectedDataLake == i {
				cursor = "> "
				lineStyle = selectedLineStyle
			}
			line := cursor + lake
			s += lineStyle.Render(line) + "\n"
		}
		s += "\nUse Up/Down arrows to navigate, 'Enter' to select."
		return s
	}

	if m.inQueryEditor {
		s += m.queryEditor.View()
		s += "\n\nPress 'esc' to return to the data lake selection.\n\n"
		if m.queryResult != "" {
			s += m.queryResult
		}

		return s
	}
	if m.confirmReset {
		s += "Are you sure you want to return to the welcome screen and discard unsaved changes? (y/n)"
		return s
	}

	// Check if a modal or different screen should be displayed
	if m.currentScreen != "" {
		switch m.currentScreen {
		case "about":
			s += "About pipeterm:\n"
			s += "pipeterm is a local data pipeline tool for services like Salesforce, Monday, etc.\n"
			s += "\nPress any key to return."
		case "help":
			s += "Help:\n"
			s += "Use the shortcut keys indicated in the action bar to navigate.\n"
			s += "Press 'Esc' at any time to return to the welcome screen.\n"
			s += "\nPress any key to return."
		case "pipelines":
			s += "Active Pipelines:\n"
			s += "- Pipeline 1\n" // Hardcoded for now
			s += "- Pipeline 2\n"
			s += "\nPress any key to return."
		case "pipeline_created":
			s += "Pipeline created successfully!\n"
			s += fmt.Sprintf("Script output: %s\n", m.scriptOutput)
			s += "\nPress any key to return to the welcome screen."
			// Reset the state to welcome after displaying the message
			m.state = "welcome"
			m.stage = 0
			m.inputs = []string{"", "", ""}
			m.cursorPosition = 0
		case "pipeline_error":
			s += "An error occurred while running the script.\n"
			s += fmt.Sprintf("Error: %s\n", m.scriptOutput)
			s += "\nPress any key to return to the welcome screen."
			// Reset the state to welcome after displaying the message
			m.state = "welcome"
			m.stage = 0
			m.inputs = []string{"", "", ""}
			m.cursorPosition = 0
		case "running_script":
			s += "Running the script...\n\n"
			s += m.progress.View() + "\n"
			s += "\nPress 'Esc' to cancel."
		}
		return s
	}

	// Display the welcome screen
	if m.state == "welcome" {
		// ASCII art welcome
		fig := figure.NewFigure("PIPETERM", "doom", true)
		welcomeText := fig.String()

		welcomeStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff79c6")). // Electric pink color
			Width(m.width).
			Align(lipgloss.Center)

		s += welcomeStyle.Render(welcomeText) + "\n\n"
		introText := lipgloss.NewStyle().
			Width(m.width).
			Align(lipgloss.Center)

		s += introText.Render("Welcome to pipeterm! A local data pipeline tool.\n")
		s += introText.Render("Press 'c' to create a new pipeline")
		return s
	}

	// Pipeline creation view
	if m.state == "create_pipeline" {

		promptStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF7F00")) // Orange

		grayedOutStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

		switch m.stage {
		case 0:
			s += promptStyle.Render("Name your pipeline: ") + "\n\n"
			if m.textInput != nil {
				s += m.textInput.View()
			} else {
				s += "<no text input>"
			}

			s += "\nPress 'Esc' at any time to return to the welcome screen."
		case 1:

			s += grayedOutStyle.Render("Name your pipeline: "+m.inputs[0]) + "\n\n"
			s += promptStyle.Render("Select service to pull from:\n")

			// Define styles
			selectedLineStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF7F00"))

			unselectedLineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
			s += "\n"
			for i, service := range m.services {
				cursor := "  "
				lineStyle := unselectedLineStyle
				if m.cursorPosition == i {
					cursor = "> "
					lineStyle = selectedLineStyle
				}
				line := cursor + service
				if service == "Bring Your Own Data" {
					line += " " + m.textInput.View()
				}
				s += lineStyle.Render(line) + "\n"
			}
			s += "\nUse Up/Down arrows to navigate, 'Enter' to select."
			s += "\nPress 'Esc' at any time to return to the welcome screen."
		case 2:
			s += grayedOutStyle.Render("Name your pipeline: "+m.inputs[0]) + "\n"
			s += grayedOutStyle.Render("Selected service: "+m.services[m.selectedService]) + "\n"
			s += promptStyle.Render("Select data loading type:\n")

			// Define styles
			selectedLineStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF7F00"))

			unselectedLineStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF"))
			s += "\n"
			for i, dataType := range m.dataTypes {
				cursor := "  "
				lineStyle := unselectedLineStyle
				if m.cursorPosition == i {
					cursor = "> "
					lineStyle = selectedLineStyle
				}
				line := cursor + dataType
				s += lineStyle.Render(line) + "\n"
			}
			s += "\nUse Up/Down arrows to navigate, 'Enter' to select."
			s += "\nPress 'Esc' at any time to return to the welcome screen."
		case 3:
			s += grayedOutStyle.Render("Name your pipeline: "+m.inputs[0]) + "\n"
			s += grayedOutStyle.Render("Selected service: "+m.services[m.selectedService]) + "\n"
			s += grayedOutStyle.Render("Data loading type: "+m.dataTypes[m.selectedDataType]) + "\n\n"
			s += promptStyle.Render("Review your information:\n")
			s += "\n"
			s += fmt.Sprintf("Pipeline Name: %s\n", m.inputs[0])
			s += fmt.Sprintf("Service: %s\n", m.services[m.selectedService])
			s += fmt.Sprintf("Data Loading Type: %s\n", m.dataTypes[m.selectedDataType])
			s += "\nPress 'Enter' to confirm, or 'Esc' to return to the welcome screen."
		}
	}

	return s
}
