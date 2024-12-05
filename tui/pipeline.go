package tui

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gorhill/cronexpr"
	"github.com/robfig/cron/v3"
)

type Pipeline struct {
	ID             int          `json:"id"`
	Name           string       `json:"name"`
	Status         string       `json:"status"`
	LastRun        time.Time    `json:"last_run"`
	NextRun        time.Time    `json:"next_run"`
	Healthy        bool         `json:"healthy"`
	Running        bool         `json:"running"`
	Logs           []string     `json:"logs"`
	CronExpr       string       `json:"cron_expr"`
	CronID         cron.EntryID `json:"-"`
	cron           *cron.Cron   `json:"-"`
	animation      []string     `json:"animation"`
	animIndex      int          `json:"anim_index"`
	ScriptPath     string       `json:"script_path"`
	ScriptType     string       `json:"script_type"`
	LastScriptPath string       `json:"last_script_path"`
}

type PipelineStorage struct {
	Pipelines []Pipeline `json:"pipelines"`
	NextID    int        `json:"next_id"`
}

type pipelineItem struct {
	pipeline Pipeline
}

type runPipelineMsg struct {
	ID     int
	Output string
	Error  error
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

	// Add safety check for animation index
	if p.Running && len(p.animation) > 0 && p.animIndex >= len(p.animation) {
		p.animIndex = 0
	}

	nameWidth := 20
	statusWidth := 15
	healthWidth := 10
	scheduleWidth := 20
	lastRunWidth := 25
	nextRunWidth := 25

	var statusSymbol string
	var statusColor lipgloss.Color

	if p.Running {
		if len(p.animation) > 0 {
			statusSymbol = p.animation[p.animIndex]
		} else {
			statusSymbol = "⋯" // fallback symbol if no animation frames
		}
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

func (m *PipelinesModel) SetSize(width, height int) {
	m.width = width
	m.height = height

	listHeight := height - 2
	if listHeight < 1 {
		listHeight = 1
	}

	m.list.SetSize(width, listHeight)
	m.viewport.Width = width
	m.viewport.Height = height
	m.logsViewport.Width = width
	m.logsViewport.Height = height
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

	// Restore cron jobs
	m.cron.Stop()                         // Stop existing cron
	m.cron = cron.New(cron.WithSeconds()) // Create new cron scheduler

	// Restore all scheduled pipelines
	for i, p := range m.pipelines {
		if p.CronExpr != "" {
			// Add "0 " prefix if it's a 5-field expression
			cronExpr := p.CronExpr
			if len(strings.Fields(cronExpr)) == 5 {
				cronExpr = "0 " + cronExpr
			}

			pipelineID := p.ID
			entryID, err := m.cron.AddFunc(cronExpr, func() {
				// Find the pipeline by ID
				var pipelineIndex int
				for j, pipeline := range m.pipelines {
					if pipeline.ID == pipelineID {
						pipelineIndex = j
						break
					}
				}

				_, _ = m.executePipeline(pipelineIndex)
			})

			if err == nil {
				m.pipelines[i].CronID = entryID
			}
		}
	}

	m.cron.Start() // Start the scheduler

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
	m.cron.Start()

	return m
}

func (m *PipelinesModel) startAnimation() {
	m.animationTicker = time.NewTicker(100 * time.Millisecond)
	go func() {
		for range m.animationTicker.C {
			for i, p := range m.pipelines {
				if p.Running && len(p.animation) > 0 { // Add check for animation length
					m.pipelines[i].animIndex = (p.animIndex + 1) % len(p.animation)
				}
			}
		}
	}()
}

func (m *PipelinesModel) AddPipeline(p Pipeline) {
	p.ID = m.nextID
	// Always initialize animation array
	p.animation = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	p.animIndex = 0

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
		case "q":
			// Only quit the entire app if we're in the main pipeline view
			if !m.showLogs && !m.showScheduler {
				m.SavePipelines()
				return m, nil
			}
			// Otherwise ignore 'q' in sub-views
		case "ctrl+c":
			m.SavePipelines()
			return m, tea.Quit
		case "r":
			if len(m.pipelines) > 0 {
				selectedIndex := m.list.Index()
				if !m.pipelines[selectedIndex].Running {
					return m, m.RunPipeline(selectedIndex)
				}
			}
		case "d":
			if len(m.pipelines) > 0 && !m.showLogs && !m.showScheduler {
				selectedIndex := m.list.Index()
				// Remove from cron if scheduled
				if m.pipelines[selectedIndex].CronID != 0 {
					m.cron.Remove(m.pipelines[selectedIndex].CronID)
				}
				// Remove the pipeline
				if selectedIndex < len(m.pipelines) {
					m.pipelines = append(m.pipelines[:selectedIndex], m.pipelines[selectedIndex+1:]...)
					// Update the list
					items := make([]list.Item, len(m.pipelines))
					for i, p := range m.pipelines {
						items[i] = pipelineItem{pipeline: p}
					}
					m.list.SetItems(items)
					m.SavePipelines()
				}
			}
		case "l":
			if !m.showLogs && !m.showScheduler && len(m.pipelines) > 0 {
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
			return m, nil
		case "s":
			if !m.showLogs && !m.showScheduler {
				m.showScheduler = true
			}
		case "enter":
			if m.showScheduler && m.scheduleInput != "" {
				m.schedulePipeline(m.scheduleInput)
				m.showScheduler = false
				m.scheduleInput = ""
			}
		case "backspace":
			if m.showScheduler && len(m.scheduleInput) > 0 {
				m.scheduleInput = m.scheduleInput[:len(m.scheduleInput)-1]
			}
		case "space":
			if m.showScheduler {
				m.scheduleInput += " "
			}
		default:
			// Handle text input for scheduler
			if m.showScheduler && len(msg.String()) == 1 {
				m.scheduleInput += msg.String()
			}
		}
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
	case runPipelineMsg:
		return m, nil
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

	if m.showLogs {
		var cmd tea.Cmd
		m.logsViewport, cmd = m.logsViewport.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
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

	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render("\nPress 'r' to run pipeline, 'l' for logs, 's' to schedule, 'q' to quit")

	mainStyle := lipgloss.NewStyle().
		MaxHeight(m.height).
		MaxWidth(m.width)

	return mainStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			separator,
			listView,
			footer,
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
	// Parse the cron expression and get next run time
	if expr, err := cronexpr.Parse(cronExpr); err == nil {
		next := expr.Next(time.Now())
		return next.Format("2006-01-02 15:04:05")
	}
	return cronExpr // Fallback to showing expression if parsing fails
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
		"Schedule Pipeline\n\n"+
			"Enter cron expression (e.g., '*/5 * * * *' for every 5 minutes):\n"+
			"> %s\n\n"+
			"Common formats:\n"+
			"* * * * *      - every minute\n"+
			"*/5 * * * *    - every 5 minutes\n"+
			"0 * * * *      - every hour\n"+
			"0 0 * * *      - every day at midnight\n\n"+
			"Press 'enter' to confirm or 'esc' to cancel",
		m.scheduleInput,
	)
}

func (m *PipelinesModel) executePipeline(index int) (string, error) {
	pipeline := m.pipelines[index]

	// Update pipeline status
	m.pipelines[index].Running = true
	m.pipelines[index].Status = "Running"
	m.SavePipelines()

	var cmd *exec.Cmd
	if pipeline.ScriptType == "byod" {
		cmd = exec.Command("python3", "/Users/brettfloyd/pipeterm/utils/byod.py", pipeline.ScriptPath)
	} else {
		cmd = exec.Command("python3", "/Users/brettfloyd/pipeterm/utils/salesforce.py")
	}

	output, err := cmd.CombinedOutput()

	// Update pipeline status based on execution result
	m.pipelines[index].Running = false
	m.pipelines[index].LastRun = time.Now()

	if err != nil {
		m.pipelines[index].Status = "Failed"
		m.pipelines[index].Healthy = false
		m.pipelines[index].Logs = append(m.pipelines[index].Logs,
			fmt.Sprintf("[%s] Pipeline execution failed: %v",
				time.Now().Format("2006-01-02 15:04:05"),
				err))
	} else {
		m.pipelines[index].Status = "Completed"
		m.pipelines[index].Healthy = true
		m.pipelines[index].Logs = append(m.pipelines[index].Logs,
			fmt.Sprintf("[%s] Pipeline executed successfully",
				time.Now().Format("2006-01-02 15:04:05")))
	}

	// Save updated pipeline state
	m.SavePipelines()

	return string(output), err
}

func (m *PipelinesModel) RunPipeline(index int) tea.Cmd {
	return func() tea.Msg {
		output, err := m.executePipeline(index)
		return runPipelineMsg{
			ID:     m.pipelines[index].ID,
			Output: output,
			Error:  err,
		}
	}
}
func (m *PipelinesModel) schedulePipeline(cronExpr string) {
	selectedIndex := m.list.Index()

	// Ensure fields are correct
	fields := strings.Fields(cronExpr)
	if len(fields) == 5 {
		cronExpr = "0 " + cronExpr
	}

	// Remove existing schedule if any
	if m.pipelines[selectedIndex].CronID != 0 {
		m.cron.Remove(m.pipelines[selectedIndex].CronID)
	}

	// Create a closure that captures the pipeline by ID rather than index
	pipelineID := m.pipelines[selectedIndex].ID

	entryID, err := m.cron.AddFunc(cronExpr, func() {
		// Find the pipeline by ID to handle reordering
		var pipelineIndex int
		found := false
		for i, p := range m.pipelines {
			if p.ID == pipelineID {
				pipelineIndex = i
				found = true
				break
			}
		}

		if !found {
			// Pipeline no longer exists
			return
		}

		// Add log entry before execution
		m.pipelines[pipelineIndex].Logs = append(m.pipelines[pipelineIndex].Logs,
			fmt.Sprintf("[%s] Cron trigger: Starting pipeline execution",
				time.Now().Format("2006-01-02 15:04:05")))

		output, err := m.executePipeline(pipelineIndex)

		// Log the execution result
		if err != nil {
			m.pipelines[pipelineIndex].Logs = append(m.pipelines[pipelineIndex].Logs,
				fmt.Sprintf("[%s] Cron execution failed: %v\nOutput: %s",
					time.Now().Format("2006-01-02 15:04:05"),
					err, output))
		} else {
			m.pipelines[pipelineIndex].Logs = append(m.pipelines[pipelineIndex].Logs,
				fmt.Sprintf("[%s] Cron execution completed successfully\nOutput: %s",
					time.Now().Format("2006-01-02 15:04:05"),
					output))
		}

		// Update next run time
		if schedule, err := cron.ParseStandard(cronExpr); err == nil {
			m.pipelines[pipelineIndex].NextRun = schedule.Next(time.Now())
		}

		// Save changes
		m.SavePipelines()
	})

	if err != nil {
		m.pipelines[selectedIndex].Logs = append(m.pipelines[selectedIndex].Logs,
			fmt.Sprintf("[%s] Failed to schedule pipeline: %v",
				time.Now().Format("2006-01-02 15:04:05"),
				err))
		return
	}

	// Store the user-friendly version in CronExpr
	if len(fields) == 5 {
		m.pipelines[selectedIndex].CronExpr = cronExpr[2:]
	} else {
		m.pipelines[selectedIndex].CronExpr = cronExpr
	}
	m.pipelines[selectedIndex].CronID = entryID

	// Set initial next run time
	if schedule, err := cron.ParseStandard(cronExpr); err == nil {
		m.pipelines[selectedIndex].NextRun = schedule.Next(time.Now())
	}

	// Add log entry with human-readable format
	m.pipelines[selectedIndex].Logs = append(m.pipelines[selectedIndex].Logs,
		fmt.Sprintf("[%s] Pipeline scheduled: %s\n%s",
			time.Now().Format("2006-01-02 15:04:05"),
			m.pipelines[selectedIndex].CronExpr,
			formatCronToHuman(m.pipelines[selectedIndex].CronExpr)))

	m.SavePipelines()
}
func formatCronToHuman(cronExpr string) string {
	expr, err := cronexpr.Parse(cronExpr)
	if err != nil {
		return "Invalid cron expression"
	}

	next := expr.Next(time.Now())
	nextAfter := expr.Next(next)

	return fmt.Sprintf("Next runs: %s, then %s",
		next.Format("Mon Jan 2 15:04:05"),
		nextAfter.Format("Mon Jan 2 15:04:05"))
}

func (m *PipelinesModel) Init() tea.Cmd {
	return nil
}
