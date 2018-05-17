package task

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/aptly-dev/aptly/aptly"
)

// Output represents a safe standard output of task
// which is compatbile to AptlyProgress.
type Output struct {
	mu     *sync.Mutex
	output *bytes.Buffer
}

// PublishOutput specific output for publishing api
type PublishOutput struct {
	*Output
	PublishDetail
	barType *aptly.BarType
}

// NewOutput creates new output
func NewOutput() *Output {
	return &Output{mu: &sync.Mutex{}, output: &bytes.Buffer{}}
}

func (t *Output) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.output.String()
}

// Write is used to determine how many bytes have been written
// not needed in our case.
func (t *Output) Write(p []byte) (n int, err error) {
	return len(p), err
}

// WriteString writes string to output
func (t *Output) WriteString(s string) (n int, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.output.WriteString(s)
}

// Start is needed for progress compatibility
func (t *Output) Start() {
	// Not implemented
}

// Shutdown is needed for progress compatibility
func (t *Output) Shutdown() {
	// Not implemented
}

// Flush is needed for progress compatibility
func (t *Output) Flush() {
	// Not implemented
}

// InitBar is needed for progress compatibility
func (t *Output) InitBar(count int64, isBytes bool, barType aptly.BarType) {
	// Not implemented
}

// InitBar publish output specific
func (t *PublishOutput) InitBar(count int64, isBytes bool, barType aptly.BarType) {
	t.barType = &barType
	if barType == aptly.BarPublishGeneratePackageFiles {
		t.TotalNumberOfPackages = count
		t.RemainingNumberOfPackages = count
		t.Store(t)
	}
}

// ShutdownBar is needed for progress compatibility
func (t *Output) ShutdownBar() {
	// Not implemented
}

// ShutdownBar publish output specific
func (t *PublishOutput) ShutdownBar() {
	t.barType = nil
}

// AddBar is needed for progress compatibility
func (t *Output) AddBar(count int) {
	// Not implemented
}

// AddBar publish output specific
func (t *PublishOutput) AddBar(count int) {
	if t.barType != nil && *t.barType == aptly.BarPublishGeneratePackageFiles {
		t.RemainingNumberOfPackages--
		t.Store(t)
	}
}

// SetBar sets current position for progress bar
func (t *Output) SetBar(count int) {
	// Not implemented
}

// Printf does printf in a safe manner
func (t *Output) Printf(msg string, a ...interface{}) {
	t.WriteString(fmt.Sprintf(msg, a...))
}

// Print does printf in a safe manner
func (t *Output) Print(msg string) {
	t.WriteString(msg)
}

// ColoredPrintf does printf in a safe manner + newline
// currently are no colors supported.
func (t *Output) ColoredPrintf(msg string, a ...interface{}) {
	t.WriteString(fmt.Sprintf(msg+"\n", a...))
}

// PrintfStdErr does printf but in safe manner to output
func (t *Output) PrintfStdErr(msg string, a ...interface{}) {
	t.WriteString(msg)
}
