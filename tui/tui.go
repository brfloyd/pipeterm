package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/common-nighthawk/go-figure"
	"os/exec"
)

// Model represents the state of the UI (exported)
type Model struct {
	state            string
	stage            int
	pipelineName     string
	services         []string
	selectedService  int
	dataTypes        []string
	selectedDataType int
	cursorPosition   int
	inputs           []string
	currentScreen    string
	confirmReset     bool
}

func runSalesforceIngestion() (string, error) {
	// Execute the Python script
	cmd := exec.Command("python3", "/Users/brettfloyd/pipeterm/utils/salesforce.py")

	// Capture combined stdout and stderr
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Return the error and output for logging or display purposes
		return string(output), err
	}

	// Return the successful script output
	return string(output), nil
}

// If successful, print the script output
func InitialModel() Model {
	return Model{
		//Start with the welcome screen when booting up the tool
		state:            "welcome",
		services:         []string{"Salesforce", "Monday", "HubSpot"},
		dataTypes:        []string{"All Data", "Batch"},
		selectedService:  0,
		selectedDataType: 0,
		cursorPosition:   0,
		inputs:           []string{"", "", ""},
		currentScreen:    "",
		confirmReset:     false,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		_, _ = msg.Width, msg.Height

	case tea.KeyMsg:

		if m.confirmReset {
			switch msg.String() {
			case "y", "Y":
				m.confirmReset = false
				m.currentScreen = ""
				m.stage = 0
				m.cursorPosition = 0
				m.inputs = []string{"", "", ""}
				m.state = "welcome"
			case "n", "N":
				m.confirmReset = false
			}
			return m, nil
		}

		// Handle Escape key to return home or quit
		if msg.Type == tea.KeyEsc {
			if m.state != "welcome" {
				m.confirmReset = true
			} else {
				return m, tea.Quit
			}
			return m, nil
		}

		// During text input stages, process keys as input characters
		if m.state == "create_pipeline" && m.stage == 0 {
			switch msg.Type {
			case tea.KeyEnter:
				if len(m.inputs[0]) > 0 {
					m.stage++
					m.cursorPosition = 0
				}
			case tea.KeyBackspace, tea.KeyDelete:
				if len(m.inputs[0]) > 0 {
					m.inputs[0] = m.inputs[0][:len(m.inputs[0])-1]
				}
			case tea.KeyRunes:
				m.inputs[0] += msg.String()
			default:

			}
			return m, nil
		}

		// Handle action bar shortcuts
		if m.currentScreen != "" {

			switch msg.String() {
			case "?":
				m.currentScreen = "help"
			case "a":
				m.currentScreen = "about"
			case "p":
				m.currentScreen = "pipelines"
			case "s":
				m.currentScreen = "save"
			case "q", "ctrl+c", "ctrl+q":
				return m, tea.Quit
			case "c":
				m.state = "create_pipeline"
				m.stage = 0
				m.currentScreen = ""
			default:
				// This causes any other key that is presses will exit the screen
				m.currentScreen = ""
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "ctrl+q", "q":
			return m, tea.Quit

		case "?":
			m.currentScreen = "help"
			return m, nil

		case "a":
			m.currentScreen = "about"
			return m, nil

		case "p":
			m.currentScreen = "pipelines"
			return m, nil

		case "s":
			m.currentScreen = "save"
			return m, nil

		case "c":
			m.state = "create_pipeline"
			m.stage = 0
			return m, nil
		}

		// Handle input based on the current state
		if m.state == "welcome" {

			if msg.Type == tea.KeyEnter {
				m.state = "create_pipeline"
				m.stage = 0
			}
		} else if m.state == "create_pipeline" {

			switch m.stage {
			case 1:
				switch msg.String() {
				case "up":
					if m.cursorPosition > 0 {
						m.cursorPosition--
					}
				case "down":
					if m.cursorPosition < len(m.services)-1 {
						m.cursorPosition++
					}
				case "enter":
					m.selectedService = m.cursorPosition
					m.stage++
					m.cursorPosition = 0
				}
			case 2:
				switch msg.String() {
				case "up":
					if m.cursorPosition > 0 {
						m.cursorPosition--
					}
				case "down":
					if m.cursorPosition < len(m.dataTypes)-1 {
						m.cursorPosition++
					}
				case "enter":
					m.selectedDataType = m.cursorPosition
					m.stage++
				}
			case 3:
				if msg.String() == "enter" {

					m.currentScreen = "pipeline_created"
				}
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	s := ""

	// Quick Action Bar
	actionBarStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#333333")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 1)

	actionBar := actionBarStyle.Render(" [c]Create Pipeline [?]Help [a]About [p]Pipelines [s]Save [q]Quit ")
	s += actionBar + "\n\n"

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
			s += "- Pipeline 1\n" // I just hardcoded these in here for now to show the concept of the pipeline screen
			s += "- Pipeline 2\n"
			s += "\nPress any key to return."
		case "save":
			s += "Saving your progress...\n"
			//This doesent actually save anything, just a placeholder message for now
			s += "Progress saved successfully.\n"
			s += "\nPress any key to continue."
		case "pipeline_created":
			s += "Pipeline created successfully!\n"
			s += "\nPress any key to return to the welcome screen."
			// Reset the state to welcome after displaying the message
			m.state = "welcome"
			m.stage = 0
			m.inputs = []string{"", "", ""}
			m.cursorPosition = 0
		}
		return s
	}

	// Display the welcome screen
	if m.state == "welcome" {
		// This is the ascii art that will welcome the user to the tool
		fig := figure.NewFigure("PIPETERM", "doom", true)
		welcomeText := fig.String()

		welcomeStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff79c6")) // Electric pink color

		s += welcomeStyle.Render(welcomeText)

		s += "\nPress 'c' to create a new pipeline, or use the action bar shortcuts.\n"
		s += "Press 'Esc' to exit."
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
			s += promptStyle.Render("Name your pipeline: ") + m.inputs[0] + "\n"
			s += "\nPress 'Esc' at any time to return to the welcome screen."
		case 1:
			s += grayedOutStyle.Render("Name your pipeline: "+m.inputs[0]) + "\n\n"
			s += promptStyle.Render("Select service to pull from:\n")

			// Define styles
			selectedLineStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF7F00"))

			unselectedLineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))

			for i, service := range m.services {
				cursor := "  "
				lineStyle := unselectedLineStyle
				if m.cursorPosition == i {
					cursor = "> "
					lineStyle = selectedLineStyle
				}
				line := cursor + service
				s += lineStyle.Render(line) + "\n"
			}
			s += "\nUse Up/Down arrows to navigate, 'Enter' to select." //TODO: Add vim jkli support
			s += "\nPress 'Esc' at any time to return to the welcome screen."
		case 2:
			s += grayedOutStyle.Render("Name your pipeline: "+m.inputs[0]) + "\n"
			s += grayedOutStyle.Render("Selected service: "+m.services[m.selectedService]) + "\n\n"
			s += promptStyle.Render("Select data loading type:\n")

			// Define styles
			selectedLineStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF7F00"))

			unselectedLineStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF"))

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
			s += fmt.Sprintf("Pipeline Name: %s\n", m.inputs[0])
			s += fmt.Sprintf("Service: %s\n", m.services[m.selectedService])
			s += fmt.Sprintf("Data Loading Type: %s\n", m.dataTypes[m.selectedDataType])
			s += "\nPress 'Enter' to confirm, or 'Esc' to return to the welcome screen."

			scriptOutput, err := runSalesforceIngestion()
			if err != nil {
				s += fmt.Sprintf("Error running script: %s\n", err)
			} else {
				s += fmt.Sprintf("Script output: %s\n", scriptOutput)
			}
		}
	}

	return s
}
