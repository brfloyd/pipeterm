package tui

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/olekukonko/tablewriter"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func runByodScriptCmd(ctx context.Context, userScriptPath string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.CommandContext(ctx, "python3", "/Users/brettfloyd/pipeterm/utils/byod.py", userScriptPath)
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

// Command to execute the query
func executeQueryCmd(dataLake string, query string) tea.Cmd {
	return func() tea.Msg {
		result, err := executeQuery(dataLake, query)
		return queryResultMsg{result: result, err: err}
	}
}

func executeQuery(dataLake string, query string) (string, error) {
	// Open a DuckDB connection
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return "", err
	}
	defer db.Close()

	// Get the user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	baseDir := filepath.Join(homeDir, ".local", "share", "pipeterm_lake", dataLake)

	// Create views for each Parquet file in the data lake
	files, err := filepath.Glob(filepath.Join(baseDir, "*.csv"))
	if err != nil {
		return "", err
	}

	for _, file := range files {
		tableName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		createViewQuery := fmt.Sprintf("CREATE VIEW %s AS SELECT * FROM read_csv('%s');", tableName, file)
		_, err := db.Exec(createViewQuery)
		if err != nil {
			return "", err
		}
	}

	// Execute the user's query
	rows, err := db.Query(query)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	// Fetch and format the results
	columns, err := rows.Columns()
	if err != nil {
		return "", err
	}

	// Prepare data for tablewriter
	var data [][]string

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return "", err
		}
		var valueStrs []string
		for _, val := range values {
			switch v := val.(type) {
			case nil:
				valueStrs = append(valueStrs, "NULL")
			case []byte:
				valueStrs = append(valueStrs, string(v))
			default:
				valueStrs = append(valueStrs, fmt.Sprintf("%v", v))
			}
		}
		data = append(data, valueStrs)
	}

	if err := rows.Err(); err != nil {
		return "", err
	}

	// Create a buffer to capture the table output
	var buf strings.Builder
	table := tablewriter.NewWriter(&buf)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(false)
	table.SetHeader(columns)
	table.AppendBulk(data)
	table.Render()

	return buf.String(), nil
}
