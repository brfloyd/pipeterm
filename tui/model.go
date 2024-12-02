package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

// Model represents the state of the UI
type Model struct {
	state             string
	stage             int
	pipelineName      string
	services          []string
	selectedService   int
	dataTypes         []string
	selectedDataType  int
	cursorPosition    int
	inputs            []string
	currentScreen     string
	confirmReset      bool
	progress          progress.Model
	progressValue     float64
	scriptOutput      string
	scriptCancel      context.CancelFunc
	dataLakes         []string
	selectedDataLake  int
	inDataLakeSelect  bool
	inQueryEditor     bool
	queryInput        string
	queryResult       string
	width             int
	height            int
	queryEditor       *QueryEditor
	textInput         *TextInputModel
	textInputActive   bool
	customServiceName string
	pipelinesModel    *PipelinesModel
	inPipelinesTab    bool
}

//Add connection to the API

func InitialModel() Model {
	dataLakes, err := listDataLakes()
	if err != nil {
		fmt.Println("Error listing data lakes:", err)
	}

	pipelinesModel := NewPipelinesModel(80, 24)

	return Model{
		// Start with the welcome screen when booting up the tool

		state:            "welcome",
		services:         []string{"Salesforce", "Monday", "HubSpot", "Bring Your Own Data"},
		dataTypes:        []string{"All Data", "Batch"},
		selectedService:  0,
		selectedDataType: 0,
		cursorPosition:   0,
		inputs:           []string{"", "", ""},
		currentScreen:    "",
		confirmReset:     false,
		progress:         progress.New(progress.WithDefaultGradient()),
		dataLakes:        dataLakes,
		inQueryEditor:    false,
		textInput:        newTextInput(),
		pipelinesModel:   pipelinesModel,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.textInput.Init(), createDataLakeFolder())
}
