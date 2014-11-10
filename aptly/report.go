package aptly

import (
	"fmt"
)

// ResultReporter is abstraction for result reporting from complex processing functions
type ResultReporter interface {
	// Warning is non-fatal error message
	Warning(msg string, a ...interface{})
	// Removed is signal that something has been removed
	Removed(msg string, a ...interface{})
	// Added is signal that something has been added
	Added(msg string, a ...interface{})
}

// ConsoleResultReporter is implementation of ResultReporter that prints in colors to console
type ConsoleResultReporter struct {
	Progress Progress
}

// Check interface
var (
	_ ResultReporter = &ConsoleResultReporter{}
)

// Warning is non-fatal error message (yellow)
func (c *ConsoleResultReporter) Warning(msg string, a ...interface{}) {
	c.Progress.ColoredPrintf("@y[!]@| @!"+msg+"@|", a...)
}

// Removed is signal that something has been removed (red)
func (c *ConsoleResultReporter) Removed(msg string, a ...interface{}) {
	c.Progress.ColoredPrintf("@r[-]@| "+msg, a...)
}

// Added is signal that something has been added (green)
func (c *ConsoleResultReporter) Added(msg string, a ...interface{}) {
	c.Progress.ColoredPrintf("@g[+]@| "+msg, a...)
}

// RecordingResultReporter is implementation of ResultReporter that collects all messages
type RecordingResultReporter struct {
	Warnings []string `json:"warnings"`
	Adds     []string `json:"added"`
	Removes  []string `json:"removed"`
}

// Check interface
var (
	_ ResultReporter = &RecordingResultReporter{}
)

// Warning is non-fatal error message
func (r *RecordingResultReporter) Warning(msg string, a ...interface{}) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(msg, a...))
}

// Removed is signal that something has been removed
func (r *RecordingResultReporter) Removed(msg string, a ...interface{}) {
	r.Removes = append(r.Removes, fmt.Sprintf(msg, a...))
}

// Added is signal that something has been added
func (r *RecordingResultReporter) Added(msg string, a ...interface{}) {
	r.Adds = append(r.Adds, fmt.Sprintf(msg, a...))
}
