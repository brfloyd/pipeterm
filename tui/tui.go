package tui

import (
	"context"
	//"database/sql"
	"fmt"
	_ "github.com/marcboeker/go-duckdb"
	"os"
	"os/exec"
	"path/filepath"
	//"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/common-nighthawk/go-figure"
)

// Model represents the state of the UI
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
	progress         progress.Model
	progressValue    float64
	scriptOutput     string
	scriptCancel     context.CancelFunc
	dataLakes        []string
	selectedDataLake int
	inDataLakeSelect bool
	inQueryEditor    bool
	queryInput       string
	queryResult      string
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

func InitialModel() Model {
	dataLakes, err := listDataLakes()
	if err != nil {
		fmt.Println("Error listing data lakes:", err)
	}
	return Model{
		// Start with the welcome screen when booting up the tool

		state:            "welcome",
		services:         []string{"Salesforce", "Monday", "HubSpot"},
		dataTypes:        []string{"All Data", "Batch"},
		selectedService:  0,
		selectedDataType: 0,
		cursorPosition:   0,
		inputs:           []string{"", "", ""},
		currentScreen:    "",
		confirmReset:     false,
		progress:         progress.New(progress.WithDefaultGradient()),
		dataLakes:        dataLakes,
	}
}

func (m Model) Init() tea.Cmd {
	return createDataLakeFolder()
}

// Message types for progress updates and script execution
type progressMsg float64
type scriptSuccessMsg string
type scriptErrorMsg struct{ err error }
type createDataLakeErrorMsg struct{ err error }
type createDataLakeSuccessMsg string

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		_, _ = msg.Width, msg.Height

	case progress.FrameMsg:
		var cmd tea.Cmd
		var newModel tea.Model
		newModel, cmd = m.progress.Update(msg)
		m.progress = newModel.(progress.Model) // Type assertion to progress.Model
		return m, cmd
	case progressMsg:
		m.progressValue += float64(msg)
		if m.progressValue > 1.0 {
			m.progressValue = 1.0
		}
		cmd := m.progress.SetPercent(m.progressValue)
		if m.progressValue < 1.0 && m.currentScreen == "running_script" {
			return m, tea.Batch(cmd, incrementProgressCmd())
		}
		return m, cmd

	case scriptSuccessMsg:
		m.scriptCancel = nil
		m.currentScreen = "pipeline_created"
		m.scriptOutput = string(msg)
		m.progressValue = 1.0
		cmd := m.progress.SetPercent(1.0)
		return m, cmd

	case scriptErrorMsg:
		m.scriptCancel = nil
		m.currentScreen = "pipeline_error"
		m.scriptOutput = msg.err.Error()
		m.progressValue = 1.0
		cmd := m.progress.SetPercent(1.0)
		return m, cmd

	case createDataLakeSuccessMsg:
		return m, nil

	case createDataLakeErrorMsg:
		return m, nil

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
				if m.currentScreen == "running_script" {
					if m.scriptCancel != nil {
						m.scriptCancel()
					}
					m.currentScreen = "welcome"
					m.state = "welcome"
					m.stage = 0
					m.inputs = []string{"", "", ""}
					m.cursorPosition = 0
					return m, nil
				}
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
			case "e":
				m.currentScreen = "query editor"
				m.inDataLakeSelect = true
				m.selectedDataLake = 0
				return m, nil

			default:
				// This causes any other key that is pressed to exit the screen
				m.currentScreen = ""
			}
			return m, nil
		}
		// Handle data lake selection
		if m.inDataLakeSelect {
			switch msg.String() {
			case "up":
				if m.selectedDataLake > 0 {
					m.selectedDataLake--
				}
			case "down":
				if m.selectedDataLake < len(m.dataLakes)-1 {
					m.selectedDataLake++
				}
			case "enter":
				m.inDataLakeSelect = false
				m.inQueryEditor = true
			case "esc":
				m.inDataLakeSelect = false
			}
			return m, nil
		}
		// Handle key messages when in query editor
		if m.inQueryEditor {
			switch msg.Type {
			case tea.KeyEnter:
				// Execute the query
				return m, nil //executeQueryCmd(m.dataLakes[m.selectedDataLake], m.queryInput)
			case tea.KeyBackspace, tea.KeyDelete:
				if len(m.queryInput) > 0 {
					m.queryInput = m.queryInput[:len(m.queryInput)-1]
				}
			case tea.KeyRunes:
				m.queryInput += msg.String()
			case tea.KeyEsc:
				m.inQueryEditor = false
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

		case "e":
			m.currentScreen = "query editor"
			m.inDataLakeSelect = true
			m.selectedDataLake = 0
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
					m.currentScreen = "running_script"
					m.progressValue = 0.0
					m.progress.SetPercent(0.0)
					// Create a context to cancel the script if needed
					var ctx context.Context
					ctx, m.scriptCancel = context.WithCancel(context.Background())
					// Start the script and progress bar
					return m, tea.Batch(runScriptCmd(ctx), incrementProgressCmd())
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

	actionBar := actionBarStyle.Render(" [c]Create Pipeline [?]Help [a]About [p]Pipelines [e]Query Editor [s]Save [q]Quit ")
	s += actionBar + "\n\n"
	if m.inDataLakeSelect {
		s += "Select a Data Lake:\n"
		for i, lake := range m.dataLakes {
			cursor := "  "
			if m.selectedDataLake == i {
				cursor = "> "
			}
			s += fmt.Sprintf("%s%s\n", cursor, lake)
		}
		s += "\nUse Up/Down arrows to navigate, 'Enter' to select."
		return s
	}

	if m.inQueryEditor {
		s += fmt.Sprintf("Data Lake: %s\n", m.dataLakes[m.selectedDataLake])
		s += "Enter your SQL query below:\n\n"
		s += m.queryInput + "\n"
		s += "\nPress 'Enter' to execute, 'Esc' to go back."
		if m.queryResult != "" {
			s += "\n\nQuery Result:\n" + m.queryResult
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
		case "query editor":
			s += "Query Editor:\n"
			s += "Enter your query here.\n"
			s += "\nPress any key to return."

		case "pipelines":
			s += "Active Pipelines:\n"
			s += "- Pipeline 1\n" // Hardcoded for now
			s += "- Pipeline 2\n"
			s += "\nPress any key to return."
		case "save":
			s += "Saving your progress...\n"
			s += "Progress saved successfully.\n"
			s += "\nPress any key to continue."
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
			s += "\nUse Up/Down arrows to navigate, 'Enter' to select."
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
		}
	}

	return s
}

// Command to run the script
func runScriptCmd(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.CommandContext(ctx, "python3", "/Users/brettfloyd/pipeterm/utils/salesforce.py")
		output, err := cmd.CombinedOutput()
		if ctx.Err() == context.Canceled {
			return scriptErrorMsg{err: fmt.Errorf("script canceled")}
		}
		if err != nil {
			return scriptErrorMsg{err: err}
		}
		return scriptSuccessMsg(string(output))
	}
}

func createDataLakeFolder() tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("python3", "/Users/brettfloyd/pipeterm/utils/create_pipeterm_lake.py")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return createDataLakeErrorMsg{err: err}
		}
		return createDataLakeSuccessMsg(string(output))
	}
}

// Command to increment the progress bar
func incrementProgressCmd() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(time.Millisecond * 100)
		return progressMsg(0.02)
	}
}

func listDataLakes() ([]string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	baseDir := filepath.Join(homeDir, ".local", "share", "pipeterm_lake")
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}
	var dataLakes []string
	for _, entry := range entries {
		if entry.IsDir() {
			dataLakes = append(dataLakes, entry.Name())
		}
	}
	return dataLakes, nil
}
