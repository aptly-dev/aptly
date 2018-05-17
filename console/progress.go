package console

import (
	"fmt"
	"os"
	"strings"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/cheggaaa/pb"
	"github.com/wsxiaoys/terminal/color"
)

const (
	codePrint = iota
	codePrintStdErr
	codeProgress
	codeHideProgress
	codeStop
	codeFlush
	codeBarEnabled
	codeBarDisabled
)

type printTask struct {
	code    int
	message string
	reply   chan bool
}

// Progress is a progress displaying subroutine, it allows to show download and other operations progress
// mixed with progress bar
type Progress struct {
	stopped  chan bool
	queue    chan printTask
	bar      *pb.ProgressBar
	barShown bool
}

// Check interface
var (
	_ aptly.Progress = (*Progress)(nil)
)

// NewProgress creates new progress instance
func NewProgress() *Progress {
	return &Progress{
		stopped: make(chan bool),
		queue:   make(chan printTask, 100),
	}
}

// Start makes progress start its work
func (p *Progress) Start() {
	go p.worker()
}

// Shutdown shuts down progress display
func (p *Progress) Shutdown() {
	p.ShutdownBar()
	p.queue <- printTask{code: codeStop}
	<-p.stopped
}

// Flush waits for all queued messages to be displayed
func (p *Progress) Flush() {
	ch := make(chan bool)
	p.queue <- printTask{code: codeFlush, reply: ch}
	<-ch
}

// InitBar starts progressbar for count bytes or count items
func (p *Progress) InitBar(count int64, isBytes bool, barType aptly.BarType) {
	if p.bar != nil {
		panic("bar already initialized")
	}
	if RunningOnTerminal() {
		p.bar = pb.New(0)
		p.bar.Total = count
		p.bar.NotPrint = true
		p.bar.Callback = func(out string) {
			p.queue <- printTask{code: codeProgress, message: out}
		}

		if isBytes {
			p.bar.SetUnits(pb.U_BYTES)
			p.bar.ShowSpeed = true
		}

		p.queue <- printTask{code: codeBarEnabled}
		p.bar.Start()
	}
}

// ShutdownBar stops progress bar and hides it
func (p *Progress) ShutdownBar() {
	if p.bar == nil {
		return
	}
	p.bar.Finish()
	p.queue <- printTask{code: codeBarDisabled}
	p.bar = nil
	p.queue <- printTask{code: codeHideProgress}
}

// Write is implementation of io.Writer to support updating of progress bar
func (p *Progress) Write(s []byte) (int, error) {
	if p.bar != nil {
		p.bar.Add(len(s))
	}
	return len(s), nil
}

// AddBar increments progress for progress bar
func (p *Progress) AddBar(count int) {
	if p.bar != nil {
		p.bar.Add(count)
	}
}

// SetBar sets current position for progress bar
func (p *Progress) SetBar(count int) {
	if p.bar != nil {
		p.bar.Set(count)
	}
}

// Printf does printf but in safe manner: not overwriting progress bar
func (p *Progress) Printf(msg string, a ...interface{}) {
	p.queue <- printTask{code: codePrint, message: fmt.Sprintf(msg, a...)}
}

// PrintfStdErr does printf but in safe manner to stderr
func (p *Progress) PrintfStdErr(msg string, a ...interface{}) {
	p.queue <- printTask{code: codePrintStdErr, message: fmt.Sprintf(msg, a...)}
}

// ColoredPrintf does printf in colored way + newline
func (p *Progress) ColoredPrintf(msg string, a ...interface{}) {
	if RunningOnTerminal() {
		p.queue <- printTask{code: codePrint, message: color.Sprintf(msg, a...) + "\n"}
	} else {
		// stip color marks
		var inColorMark, inCurly bool
		msg = strings.Map(func(r rune) rune {
			if inColorMark {
				if inCurly {
					if r == '}' {
						inCurly = false
						inColorMark = false
						return -1
					}
				} else {
					if r == '{' {
						inCurly = true
					} else if r == '@' {
						return '@'
					} else {
						inColorMark = false
					}
				}
				return -1
			}

			if r == '@' {
				inColorMark = true
				return -1
			}

			return r
		}, msg)

		p.Printf(msg+"\n", a...)
	}
}

func (p *Progress) worker() {
	hasBar := false

	for {
		task := <-p.queue
		switch task.code {
		case codeBarEnabled:
			hasBar = true
		case codeBarDisabled:
			hasBar = false
		case codePrint:
			if p.barShown {
				fmt.Print("\r\033[2K")
				p.barShown = false
			}
			fmt.Print(task.message)
		case codePrintStdErr:
			if p.barShown {
				fmt.Print("\r\033[2K")
				p.barShown = false
			}
			fmt.Fprint(os.Stderr, task.message)
		case codeProgress:
			if hasBar {
				fmt.Print("\r" + task.message)
				p.barShown = true
			}
		case codeHideProgress:
			if p.barShown {
				fmt.Print("\r\033[2K")
				p.barShown = false
			}
		case codeFlush:
			task.reply <- true
		case codeStop:
			p.stopped <- true
			return
		}
	}
}
