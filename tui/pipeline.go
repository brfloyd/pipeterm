package tui

import (
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	//"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/robfig/cron/v3"
)

// Pipeline represents a data pipeline.
type Pipeline struct {
	Name      string
	Status    string
	LastRun   time.Time
	NextRun   time.Time
	Healthy   bool
	Running   bool
	Logs      []string
	CronExpr  string
	CronID    cron.EntryID
	cron      *cron.Cron
	animation []string
	animIndex int
}

// pipelineItem is a custom list item.
type pipelineItem struct {
	pipeline Pipeline
}

func (i pipelineItem) Title() string {
	return i.pipeline.Name
}

func (i pipelineItem) Description() string {
	return i.pipeline.Status
}

func (i pipelineItem) FilterValue() string {
	return i.pipeline.Name
}

// pipelineDelegate renders pipeline items in the list.
type pipelineDelegate struct{}

func (d pipelineDelegate) Height() int                               { return 1 }
func (d pipelineDelegate) Spacing() int                              { return 0 }
func (d pipelineDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d pipelineDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(pipelineItem)
	if !ok {
		return
	}

	p := item.pipeline

	// Determine the status symbol and color
	var statusSymbol string
	var statusColor lipgloss.Color

	if p.Running {
		statusSymbol = p.animation[p.animIndex]
		statusColor = lipgloss.Color("5") // Purple
	} else if p.Healthy {
		statusSymbol = "✔"
		statusColor = lipgloss.Color("2") // Green
	} else {
		statusSymbol = "✖"
		statusColor = lipgloss.Color("1") // Red
	}

	// Apply styling
	nameStyle := lipgloss.NewStyle().Foreground(statusColor)
	line := nameStyle.Render(fmt.Sprintf("%s %s", statusSymbol, p.Name))

	// Highlight selection
	if index == m.Index() {
		selectedStyle := lipgloss.NewStyle().Background(lipgloss.Color("7"))
		line = selectedStyle.Render(line)
	}

	fmt.Fprintln(w, line)
}

// PipelinesModel handles the state and logic for the pipelines tab.
type PipelinesModel struct {
	pipelines       []Pipeline
	list            list.Model
	viewport        viewport.Model
	width, height   int
	selectedIndex   int
	showScheduler   bool
	scheduleInput   string
	animationTicker *time.Ticker
	healthTicker    *time.Ticker
	cron            *cron.Cron
}

// NewPipelinesModel initializes a new PipelinesModel.
func NewPipelinesModel(width, height int) *PipelinesModel {
	m := &PipelinesModel{
		pipelines: make([]Pipeline, 0),
		width:     width,
		height:    height,
		cron:      cron.New(cron.WithSeconds()),
	}

	// Initialize the list component with custom delegate.
	m.list = list.New(nil, pipelineDelegate{}, width/2, height)
	m.list.Title = "Pipelines"

	// Initialize the viewport component.
	m.viewport = viewport.New(width/2, height)

	// Start background tasks.
	m.startAnimation()
	m.startHealthChecks()

	return m
}

// startAnimation initiates the spinner animation for running pipelines.
func (m *PipelinesModel) startAnimation() {
	m.animationTicker = time.NewTicker(100 * time.Millisecond)
	go func() {
		for range m.animationTicker.C {
			for i, p := range m.pipelines {
				if p.Running {
					m.pipelines[i].animIndex = (p.animIndex + 1) % len(p.animation)
				}
			}
		}
	}()
}

// startHealthChecks initiates periodic health checks for the pipelines.
func (m *PipelinesModel) startHealthChecks() {
	m.healthTicker = time.NewTicker(30 * time.Second)
	go func() {
		for range m.healthTicker.C {
			m.checkPipelinesHealth()
		}
	}()
}

// checkPipelinesHealth updates the health status of each pipeline.
func (m *PipelinesModel) checkPipelinesHealth() {
	for i, p := range m.pipelines {
		healthy := checkPipelineAPI(p)
		m.pipelines[i].Healthy = healthy
	}
}

// checkPipelineAPI simulates an API call to check pipeline health.
func checkPipelineAPI(p Pipeline) bool {
	// Implement your actual API call here.
	return true // Placeholder
}

// Update handles incoming messages and updates the model accordingly.
func (m *PipelinesModel) Update(msg tea.Msg) (*PipelinesModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "s":
			// Open schedule input.
			m.showScheduler = true
		case "enter":
			if m.showScheduler {
				m.schedulePipeline(m.scheduleInput)
				m.showScheduler = false
				m.scheduleInput = ""
			}
		case "up", "k":
			m.list.CursorUp()
		case "down", "j":
			m.list.CursorDown()
		}
	}

	// Update list and viewport.
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the UI components.
func (m *PipelinesModel) View() string {
	if m.showScheduler {
		return m.renderScheduler()
	}
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.renderPipelineList(),
		m.renderPipelineDetails(),
	)
}

// renderPipelineList renders the list of pipelines.
func (m *PipelinesModel) renderPipelineList() string {
	items := make([]list.Item, len(m.pipelines))
	for i, p := range m.pipelines {
		items[i] = m.pipelineListItem(p)
	}
	m.list.SetItems(items)
	return m.list.View()
}

// pipelineListItem creates a list item for a pipeline.
func (m *PipelinesModel) pipelineListItem(p Pipeline) list.Item {
	return pipelineItem{pipeline: p}
}

// renderPipelineDetails shows details of the selected pipeline.
func (m *PipelinesModel) renderPipelineDetails() string {
	if len(m.pipelines) == 0 {
		return "No pipelines available."
	}
	p := m.pipelines[m.list.Index()]

	// Create a styled title for the viewport.
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("5")) // Adjust the color as needed.

	title := titleStyle.Render("Pipeline Details")

	details := fmt.Sprintf(
		"%s\n\nName: %s\nStatus: %s\nLast Run: %s\nNext Run: %s\nLogs:\n%s",
		title,
		p.Name,
		p.Status,
		p.LastRun.Format(time.RFC1123),
		p.NextRun.Format(time.RFC1123),
		formatLogs(p.Logs),
	)

	m.viewport.SetContent(details)
	return m.viewport.View()
}

// formatLogs formats the logs for display.
func formatLogs(logs []string) string {
	if len(logs) == 0 {
		return "No logs available."
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(
		fmt.Sprintf("%s", logs),
	)
}

// renderScheduler displays the cron scheduling input.
func (m *PipelinesModel) renderScheduler() string {
	return fmt.Sprintf(
		"Enter cron expression for scheduling:\n> %s",
		m.scheduleInput,
	)
}

// schedulePipeline schedules the selected pipeline.
func (m *PipelinesModel) schedulePipeline(cronExpr string) {
	selectedPipeline := m.pipelines[m.list.Index()]
	entryID, err := m.cron.AddFunc(cronExpr, func() {
		runPipeline(selectedPipeline)
	})
	if err != nil {
		// Handle error (e.g., show a message to the user).
		return
	}
	m.pipelines[m.list.Index()].CronExpr = cronExpr
	m.pipelines[m.list.Index()].CronID = entryID
	m.cron.Start()
}

// runPipeline executes the pipeline.
func runPipeline(p Pipeline) {
	// Implement your pipeline execution logic here.
}

// SetSize updates the dimensions of the model's components.
func (m *PipelinesModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.list.SetSize(width/2, height)
	m.viewport.Width = width / 2
	m.viewport.Height = height
}

// Init is called when the model is initialized.
func (m *PipelinesModel) Init() tea.Cmd {
	return nil
}
