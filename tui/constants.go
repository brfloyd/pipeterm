package tui

// Message types for progress updates and script execution
type progressMsg float64
type scriptSuccessMsg string
type scriptErrorMsg struct{ err error }
type createDataLakeErrorMsg struct{ err error }
type createDataLakeSuccessMsg string
