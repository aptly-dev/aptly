package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/smira/commander"
	"github.com/smira/flag"
	. "gopkg.in/check.v1"
)

type TaskRunSuite struct {
	cmd          *commander.Command
	mockProgress *MockTaskRunProgress
	mockContext  *MockTaskRunContext
	tempFile     *os.File
}

var _ = Suite(&TaskRunSuite{})

func (s *TaskRunSuite) SetUpTest(c *C) {
	s.cmd = makeCmdTaskRun()
	s.mockProgress = &MockTaskRunProgress{}

	// Set up mock context
	s.mockContext = &MockTaskRunContext{
		flags:    s.cmd.Flag,
		progress: s.mockProgress,
	}

	// Set up required flags
	s.cmd.Flag.String("filename", "", "specifies the filename that contains the commands to run")

	// Set mock context globally
	context = s.mockContext
}

func (s *TaskRunSuite) TearDownTest(c *C) {
	// Clean up temp file if created
	if s.tempFile != nil {
		os.Remove(s.tempFile.Name())
		s.tempFile = nil
	}
}

func (s *TaskRunSuite) TestMakeCmdTaskRun(c *C) {
	// Test command creation and basic properties
	cmd := makeCmdTaskRun()
	c.Check(cmd, NotNil)
	c.Check(cmd.UsageLine, Equals, "run (-filename=<filename> | <commands>...)")
	c.Check(cmd.Short, Equals, "run aptly tasks")
	c.Check(strings.Contains(cmd.Long, "Command helps organise multiple aptly commands"), Equals, true)

	// Test flags
	filenameFlag := cmd.Flag.Lookup("filename")
	c.Check(filenameFlag, NotNil)
	c.Check(filenameFlag.DefValue, Equals, "")
}

func (s *TaskRunSuite) TestAptlyTaskRunFromArgs(c *C) {
	// Test running tasks from command line arguments
	args := []string{"repo", "create", "test,", "repo", "list"}

	err := aptlyTaskRun(s.cmd, args)
	c.Check(err, IsNil)

	// Check that progress messages were displayed
	c.Check(len(s.mockProgress.ColoredMessages) > 0, Equals, true)
	foundRunningMessage := false
	for _, msg := range s.mockProgress.ColoredMessages {
		if strings.Contains(msg, "[Running]") {
			foundRunningMessage = true
			break
		}
	}
	c.Check(foundRunningMessage, Equals, true)
}

func (s *TaskRunSuite) TestAptlyTaskRunFromFileBasic(c *C) {
	// Create a temporary file with commands
	tempFile, err := os.CreateTemp("", "aptly-task-test-*.txt")
	c.Assert(err, IsNil)
	s.tempFile = tempFile

	commands := "repo create test\nrepo list\n"
	_, err = tempFile.WriteString(commands)
	c.Assert(err, IsNil)
	tempFile.Close()

	// Set filename flag
	s.cmd.Flag.Set("filename", tempFile.Name())

	err = aptlyTaskRun(s.cmd, []string{})
	c.Check(err, IsNil)

	// Check that file was read and commands executed
	foundReadingMessage := false
	for _, msg := range s.mockProgress.Messages {
		if strings.Contains(msg, "Reading file") {
			foundReadingMessage = true
			break
		}
	}
	c.Check(foundReadingMessage, Equals, true)
}

func (s *TaskRunSuite) TestAptlyTaskRunFileNotFound(c *C) {
	// Test with non-existent file
	s.cmd.Flag.Set("filename", "/nonexistent/file.txt")

	err := aptlyTaskRun(s.cmd, []string{})
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*no such file.*")
}

func (s *TaskRunSuite) TestAptlyTaskRunFileIsDirectory(c *C) {
	// Test with directory instead of file
	tempDir, err := os.MkdirTemp("", "aptly-task-test-dir-*")
	c.Assert(err, IsNil)
	defer os.RemoveAll(tempDir)

	s.cmd.Flag.Set("filename", tempDir)

	err = aptlyTaskRun(s.cmd, []string{})
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*no such file.*")
}

func (s *TaskRunSuite) TestAptlyTaskRunEmptyFile(c *C) {
	// Create an empty temporary file
	tempFile, err := os.CreateTemp("", "aptly-task-empty-*.txt")
	c.Assert(err, IsNil)
	s.tempFile = tempFile
	tempFile.Close()

	s.cmd.Flag.Set("filename", tempFile.Name())

	err = aptlyTaskRun(s.cmd, []string{})
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*the file is empty.*")
}

func (s *TaskRunSuite) TestAptlyTaskRunFileReadError(c *C) {
	// Create a file and then make it unreadable
	tempFile, err := os.CreateTemp("", "aptly-task-unreadable-*.txt")
	c.Assert(err, IsNil)
	s.tempFile = tempFile

	commands := "repo create test\n"
	_, err = tempFile.WriteString(commands)
	c.Assert(err, IsNil)
	tempFile.Close()

	// Make file unreadable
	err = os.Chmod(tempFile.Name(), 0000)
	c.Assert(err, IsNil)
	defer os.Chmod(tempFile.Name(), 0644) // Restore permissions for cleanup

	s.cmd.Flag.Set("filename", tempFile.Name())

	err = aptlyTaskRun(s.cmd, []string{})
	c.Check(err, NotNil)
}

func (s *TaskRunSuite) TestAptlyTaskRunStdinEmpty(c *C) {
	// Test stdin input with empty input
	// Mock stdin to return empty input
	originalStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	defer func() { os.Stdin = originalStdin }()

	// Close write end immediately to simulate empty input
	w.Close()

	err := aptlyTaskRun(s.cmd, []string{})
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*nothing entered.*")
}

func (s *TaskRunSuite) TestAptlyTaskRunStdinWithCommands(c *C) {
	// Test stdin input with commands
	originalStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	defer func() { os.Stdin = originalStdin }()

	// Write commands to pipe
	go func() {
		defer w.Close()
		fmt.Fprintln(w, "repo create test")
		fmt.Fprintln(w, "repo list")
		fmt.Fprintln(w, "") // Empty line to finish
	}()

	err := aptlyTaskRun(s.cmd, []string{})
	c.Check(err, IsNil)

	// Check that commands were processed
	c.Check(len(s.mockProgress.ColoredMessages) > 0, Equals, true)
}

func (s *TaskRunSuite) TestAptlyTaskRunCommandError(c *C) {
	// Test with command that will error
	s.mockContext.shouldErrorRun = true
	args := []string{"invalid", "command,", "repo", "list"}

	err := aptlyTaskRun(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*at least one command has reported an error.*")

	// Check that subsequent commands were skipped
	foundSkippingMessage := false
	for _, msg := range s.mockProgress.ColoredMessages {
		if strings.Contains(msg, "[Skipping]") {
			foundSkippingMessage = true
			break
		}
	}
	c.Check(foundSkippingMessage, Equals, true)
}

func (s *TaskRunSuite) TestAptlyTaskRunReOpenDatabaseError(c *C) {
	// Test with database reopen error
	s.mockContext.shouldErrorReOpenDB = true
	args := []string{"repo", "create", "test"}

	err := aptlyTaskRun(s.cmd, args)
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*failed to reopen DB.*")
}

func (s *TaskRunSuite) TestFormatCommandsBasic(c *C) {
	// Test basic command formatting
	args := []string{"repo", "create", "test,", "repo", "list"}
	result := formatCommands(args)

	c.Check(len(result), Equals, 2)
	c.Check(result[0], DeepEquals, []string{"repo", "create", "test"})
	c.Check(result[1], DeepEquals, []string{"repo", "list"})
}

func (s *TaskRunSuite) TestFormatCommandsNoComma(c *C) {
	// Test command formatting without comma separator
	args := []string{"repo", "create", "test"}
	result := formatCommands(args)

	c.Check(len(result), Equals, 1)
	c.Check(result[0], DeepEquals, []string{"repo", "create", "test"})
}

func (s *TaskRunSuite) TestFormatCommandsMultipleCommas(c *C) {
	// Test command formatting with multiple commands
	args := []string{"repo", "create", "test1,", "repo", "create", "test2,", "repo", "list"}
	result := formatCommands(args)

	c.Check(len(result), Equals, 3)
	c.Check(result[0], DeepEquals, []string{"repo", "create", "test1"})
	c.Check(result[1], DeepEquals, []string{"repo", "create", "test2"})
	c.Check(result[2], DeepEquals, []string{"repo", "list"})
}

func (s *TaskRunSuite) TestFormatCommandsEmptyCommand(c *C) {
	// Test command formatting with empty command (just comma)
	args := []string{",", "repo", "list"}
	result := formatCommands(args)

	c.Check(len(result), Equals, 2)
	c.Check(result[0], DeepEquals, []string{""})
	c.Check(result[1], DeepEquals, []string{"repo", "list"})
}

func (s *TaskRunSuite) TestFormatCommandsTrailingComma(c *C) {
	// Test command formatting with trailing comma
	args := []string{"repo", "create", "test,"}
	result := formatCommands(args)

	c.Check(len(result), Equals, 1)
	c.Check(result[0], DeepEquals, []string{"repo", "create", "test"})
}

func (s *TaskRunSuite) TestAptlyTaskRunFileWithQuotedArgs(c *C) {
	// Test file with quoted arguments
	tempFile, err := os.CreateTemp("", "aptly-task-quoted-*.txt")
	c.Assert(err, IsNil)
	s.tempFile = tempFile

	commands := "repo create \"test repo\"\nrepo list\n"
	_, err = tempFile.WriteString(commands)
	c.Assert(err, IsNil)
	tempFile.Close()

	s.cmd.Flag.Set("filename", tempFile.Name())

	err = aptlyTaskRun(s.cmd, []string{})
	c.Check(err, IsNil)

	// Should handle quoted arguments correctly
	c.Check(len(s.mockProgress.ColoredMessages) > 0, Equals, true)
}

func (s *TaskRunSuite) TestAptlyTaskRunProgressOutput(c *C) {
	// Test that progress output is correctly formatted
	args := []string{"repo", "create", "test,", "repo", "list"}

	err := aptlyTaskRun(s.cmd, args)
	c.Check(err, IsNil)

	// Check for specific progress message formats
	foundBeginOutput := false
	foundEndOutput := false
	for _, msg := range s.mockProgress.ColoredMessages {
		if strings.Contains(msg, "Begin command output") {
			foundBeginOutput = true
		}
		if strings.Contains(msg, "End command output") {
			foundEndOutput = true
		}
	}
	c.Check(foundBeginOutput, Equals, true)
	c.Check(foundEndOutput, Equals, true)
}

func (s *TaskRunSuite) TestAptlyTaskRunFlush(c *C) {
	// Test that progress flush is called
	args := []string{"repo", "list"}

	err := aptlyTaskRun(s.cmd, args)
	c.Check(err, IsNil)

	// Check that flush was called
	c.Check(s.mockProgress.flushCalled, Equals, true)
}

// Mock implementations for testing

type MockTaskRunProgress struct {
	Messages        []string
	ColoredMessages []string
	flushCalled     bool
}

func (m *MockTaskRunProgress) Printf(msg string, a ...interface{}) {
	formatted := fmt.Sprintf(msg, a...)
	m.Messages = append(m.Messages, formatted)
}

func (m *MockTaskRunProgress) ColoredPrintf(msg string, a ...interface{}) {
	formatted := fmt.Sprintf(msg, a...)
	m.ColoredMessages = append(m.ColoredMessages, formatted)
}

func (m *MockTaskRunProgress) Flush() {
	m.flushCalled = true
}

type MockTaskRunContext struct {
	flags               *flag.FlagSet
	progress            *MockTaskRunProgress
	shouldErrorReOpenDB bool
	shouldErrorRun      bool
}

func (m *MockTaskRunContext) Flags() *flag.FlagSet     { return m.flags }
func (m *MockTaskRunContext) Progress() aptly.Progress { return m.progress }

func (m *MockTaskRunContext) ReOpenDatabase() error {
	if m.shouldErrorReOpenDB {
		return fmt.Errorf("mock reopen database error")
	}
	return nil
}

// Run function is already defined in run.go:12

// RootCommand function is already defined in cmd.go:81

// CleanupContext function is already defined in context.go:16

// Test helper for stdin simulation
func simulateStdinInput(input string) (func(), error) {
	originalStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	os.Stdin = r

	go func() {
		defer w.Close()
		io.WriteString(w, input)
	}()

	cleanup := func() {
		os.Stdin = originalStdin
		r.Close()
	}

	return cleanup, nil
}

// Additional test for stdin input with proper mocking
func (s *TaskRunSuite) TestAptlyTaskRunStdinInputMocked(c *C) {
	// Test stdin functionality with better mocking
	cleanup, err := simulateStdinInput("repo create test\nrepo list\n\n")
	c.Assert(err, IsNil)
	defer cleanup()

	err = aptlyTaskRun(s.cmd, []string{})
	c.Check(err, IsNil)

	// Verify commands were processed
	foundRunningMessage := false
	for _, msg := range s.mockProgress.ColoredMessages {
		if strings.Contains(msg, "[Running]") {
			foundRunningMessage = true
			break
		}
	}
	c.Check(foundRunningMessage, Equals, true)
}

func (s *TaskRunSuite) TestAptlyTaskRunScannerError(c *C) {
	// Test scanner error handling
	tempFile, err := os.CreateTemp("", "aptly-task-scanner-*.txt")
	c.Assert(err, IsNil)
	s.tempFile = tempFile

	// Write some content and close file
	commands := "repo create test\n"
	_, err = tempFile.WriteString(commands)
	c.Assert(err, IsNil)
	tempFile.Close()

	// Create a mock that will simulate scanner error
	originalOpen := os.Open
	os.Open = func(name string) (*os.File, error) {
		if name == tempFile.Name() {
			// Return a file that will cause scanner issues
			return os.Open("/dev/null")
		}
		return originalOpen(name)
	}
	defer func() { os.Open = originalOpen }()

	s.cmd.Flag.Set("filename", tempFile.Name())

	err = aptlyTaskRun(s.cmd, []string{})
	// Should succeed with /dev/null but have no commands
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*the file is empty.*")
}
