package tui

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/robfig/cron/v3"
)

type Pipeline struct {
	ID        int          `json:"id"`
	Name      string       `json:"name"`
	Status    string       `json:"status"`
	LastRun   time.Time    `json:"last_run"`
	NextRun   time.Time    `json:"next_run"`
	Healthy   bool         `json:"healthy"`
	Running   bool         `json:"running"`
	Logs      []string     `json:"logs"`
	CronExpr  string       `json:"cron_expr"`
	CronID    cron.EntryID `json:"-"` // Don't store in JSON
	cron      *cron.Cron   `json:"-"` // Don't store in JSON
	animation []string     `json:"animation"`
	animIndex int          `json:"anim_index"`
}

type PipelineStorage struct {
	Pipelines []Pipeline `json:"pipelines"`
	NextID    int        `json:"next_id"`
}

type pipelineItem struct {
	pipeline Pipeline
}

func (i pipelineItem) Title() string       { return i.pipeline.Name }
func (i pipelineItem) Description() string { return i.pipeline.Status }
func (i pipelineItem) FilterValue() string { return i.pipeline.Name }

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

	nameWidth := 20
	statusWidth := 15
	healthWidth := 10
	scheduleWidth := 20
	lastRunWidth := 25
	nextRunWidth := 25

	var statusSymbol string
	var statusColor lipgloss.Color

	if p.Running {
		statusSymbol = p.animation[p.animIndex]
		statusColor = lipgloss.Color("5")
	} else if p.Healthy {
		statusSymbol = "✔"
		statusColor = lipgloss.Color("2")
	} else {
		statusSymbol = "✖"
		statusColor = lipgloss.Color("1")
	}

	baseStyle := lipgloss.NewStyle().PaddingRight(1)
	selectedStyle := lipgloss.NewStyle().Background(lipgloss.Color("7"))

	name := baseStyle.Copy().Width(nameWidth).
		Foreground(statusColor).
		Render(fmt.Sprintf("%s %s", statusSymbol, p.Name))
	status := baseStyle.Copy().Width(statusWidth).Render(p.Status)
	health := baseStyle.Copy().Width(healthWidth).Render(getBoolEmoji(p.Healthy))
	schedule := baseStyle.Copy().Width(scheduleWidth).Render(getScheduleDisplay(p.CronExpr))
	lastRun := baseStyle.Copy().Width(lastRunWidth).Render(formatTime(p.LastRun))
	nextRun := baseStyle.Copy().Width(nextRunWidth).Render(formatTime(p.NextRun))

	line := fmt.Sprintf("%s%s%s%s%s%s",
		name,
		status,
		health,
		schedule,
		lastRun,
		nextRun,
	)

	if index == m.Index() {
		line = selectedStyle.Render(line)
	}

	fmt.Fprintln(w, line)
}

type PipelinesModel struct {
	pipelines       []Pipeline
	list            list.Model
	viewport        viewport.Model
	logsViewport    viewport.Model
	width, height   int
	selectedIndex   int
	showScheduler   bool
	showLogs        bool
	scheduleInput   string
	animationTicker *time.Ticker
	healthTicker    *time.Ticker
	cron            *cron.Cron
	nextID          int
}

func getStorageDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	storageDir := filepath.Join(homeDir, ".local", "share", "pipeterm_storage")
	return storageDir, nil
}

func (m *PipelinesModel) SavePipelines() error {
	storageDir, err := getStorageDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return err
	}

	storage := PipelineStorage{
		Pipelines: m.pipelines,
		NextID:    m.nextID,
	}

	data, err := json.Marshal(storage)
	if err != nil {
		return err
	}

	pipelinePath := filepath.Join(storageDir, "pipelines.json")
	return os.WriteFile(pipelinePath, data, 0644)
}

func (m *PipelinesModel) LoadPipelines() error {
	storageDir, err := getStorageDir()
	if err != nil {
		return err
	}

	pipelinePath := filepath.Join(storageDir, "pipelines.json")

	if _, err := os.Stat(pipelinePath); os.IsNotExist(err) {
		return nil // No saved pipelines yet
	}

	data, err := os.ReadFile(pipelinePath)
	if err != nil {
		return err
	}

	var storage PipelineStorage
	if err := json.Unmarshal(data, &storage); err != nil {
		return err
	}

	m.pipelines = storage.Pipelines
	m.nextID = storage.NextID

	items := make([]list.Item, len(m.pipelines))
	for i, p := range m.pipelines {
		items[i] = pipelineItem{pipeline: p}
	}
	m.list.SetItems(items)

	return nil
}

func NewPipelinesModel(width, height int) *PipelinesModel {
	m := &PipelinesModel{
		pipelines: make([]Pipeline, 0),
		width:     width,
		height:    height,
		cron:      cron.New(cron.WithSeconds()),
		nextID:    1,
	}

	delegate := pipelineDelegate{}
	listHeight := height - 2
	if listHeight < 1 {
		listHeight = 1
	}

	m.list = list.New(nil, delegate, width, listHeight)
	m.list.SetShowTitle(false)
	m.list.SetShowStatusBar(false)
	m.list.SetFilteringEnabled(false)

	m.viewport = viewport.New(width, height)
	m.logsViewport = viewport.New(width, height)
	m.logsViewport.SetContent("press 'l' to show logs")

	// Load saved pipelines
	if err := m.LoadPipelines(); err != nil {
		fmt.Printf("Error loading pipelines: %v\n", err)
	}

	m.startAnimation()
	m.startHealthChecks()

	return m
}

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

func (m *PipelinesModel) AddPipeline(p Pipeline) {
	p.ID = m.nextID
	if p.animation == nil {
		p.animation = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	}
	if len(p.Logs) == 0 {
		p.Logs = []string{"[Pipeline Created.]"}
	}
	if p.Status == "" {
		p.Status = "Idle"
	}
	p.LastRun = time.Now()

	m.pipelines = append(m.pipelines, p)
	m.nextID++

	items := make([]list.Item, len(m.pipelines))
	for i, pipeline := range m.pipelines {
		items[i] = pipelineItem{pipeline: pipeline}
	}
	m.list.SetItems(items)

	// Save after adding
	m.SavePipelines()
}

func (m *PipelinesModel) startHealthChecks() {
	m.healthTicker = time.NewTicker(30 * time.Second)
	go func() {
		for range m.healthTicker.C {
			m.checkPipelinesHealth()
		}
	}()
}

func (m *PipelinesModel) checkPipelinesHealth() {
	for i, p := range m.pipelines {
		healthy := checkPipelineAPI(p)
		m.pipelines[i].Healthy = healthy
	}
	// Save after health check updates
	m.SavePipelines()
}

func checkPipelineAPI(p Pipeline) bool {
	return true
}

func (m *PipelinesModel) Update(msg tea.Msg) (*PipelinesModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.SavePipelines() // Save before quitting
			return m, tea.Quit
		case "l":
			if !m.showLogs && len(m.pipelines) > 0 {
				selectedPipeline := m.pipelines[m.list.Index()]
				logsContent := formatLogs(selectedPipeline.Logs)
				m.logsViewport.SetContent(logsContent)
				m.showLogs = true
			}
		case "esc":
			if m.showLogs {
				m.showLogs = false
			} else if m.showScheduler {
				m.showScheduler = false
				m.scheduleInput = ""
			}
		case "s":
			m.showScheduler = true
		case "enter":
			if m.showScheduler {
				m.schedulePipeline(m.scheduleInput)
				m.showScheduler = false
				m.scheduleInput = ""
			}
		}
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
	}

	if !m.showLogs {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *PipelinesModel) View() string {
	if m.showLogs {
		return m.renderLogsView()
	}
	if m.showScheduler {
		return m.renderScheduler()
	}

	nameWidth := 20
	statusWidth := 15
	healthWidth := 10
	scheduleWidth := 20
	lastRunWidth := 25
	nextRunWidth := 25

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12"))

	header := lipgloss.JoinHorizontal(
		lipgloss.Left,
		headerStyle.Copy().Width(nameWidth).Render("NAME"),
		headerStyle.Copy().Width(statusWidth).Render("STATUS"),
		headerStyle.Copy().Width(healthWidth).Render("HEALTH"),
		headerStyle.Copy().Width(scheduleWidth).Render("SCHEDULE"),
		headerStyle.Copy().Width(lastRunWidth).Render("LAST RUN"),
		headerStyle.Copy().Width(nextRunWidth).Render("NEXT RUN"),
	)

	separator := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render(strings.Repeat("─", m.width))

	items := make([]list.Item, len(m.pipelines))
	for i, p := range m.pipelines {
		items[i] = pipelineItem{pipeline: p}
	}
	m.list.SetItems(items)

	listView := m.list.View()

	mainStyle := lipgloss.NewStyle().
		MaxHeight(m.height).
		MaxWidth(m.width)

	return mainStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			separator,
			listView,
		),
	)
}

func (m *PipelinesModel) renderLogsView() string {
	if len(m.pipelines) == 0 {
		return "No pipelines available."
	}

	p := m.pipelines[m.list.Index()]
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("5"))

	title := titleStyle.Render(fmt.Sprintf("Logs for Pipeline: %s", p.Name))

	logsContent := fmt.Sprintf(
		"%s\n\n%s",
		title,
		formatLogs(p.Logs),
	)

	m.logsViewport.SetContent(logsContent)
	m.logsViewport.Height = m.height - 4
	m.logsViewport.Width = m.width

	return m.logsViewport.View()
}

func getBoolEmoji(b bool) string {
	if b {
		return "✅ OK"
	}
	return "❌ Error"
}

func getScheduleDisplay(cronExpr string) string {
	if cronExpr == "" {
		return "Not scheduled"
	}
	return cronExpr
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "Never"
	}
	return t.Format("2006-01-02 15:04:05")
}

func formatLogs(logs []string) string {
	if len(logs) == 0 {
		return "No logs available."
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(
		strings.Join(logs, "\n"),
	)
}

func (m *PipelinesModel) renderScheduler() string {
	return fmt.Sprintf(
		"Enter cron expression for scheduling:\n> %s",
		m.scheduleInput,
	)
}

func (m *PipelinesModel) schedulePipeline(cronExpr string) {
	selectedPipeline := m.pipelines[m.list.Index()]
	entryID, err := m.cron.AddFunc(cronExpr, func() {
		runPipeline(selectedPipeline)
	})
	if err != nil {
		return
	}
	m.pipelines[m.list.Index()].CronExpr = cronExpr
	m.pipelines[m.list.Index()].CronID = entryID
	m.cron.Start()
}

func runPipeline(p Pipeline) {
}

func (m *PipelinesModel) SetSize(width, height int) {
	m.width = width
	m.height = height

	listHeight := height - 2
	if listHeight < 1 {
		listHeight = 1
	}

	m.list.SetSize(width, listHeight)
}

func (m *PipelinesModel) Init() tea.Cmd {
	return nil
}
