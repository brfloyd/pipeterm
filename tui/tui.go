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

	tea "github.com/charmbracelet/bubbletea"
)

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
