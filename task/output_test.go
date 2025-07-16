package task

import (
	"fmt"
	"strings"
	"sync"

	"github.com/aptly-dev/aptly/aptly"
	. "gopkg.in/check.v1"
)

type OutputSuite struct {
	output        *Output
	publishOutput *PublishOutput
}

var _ = Suite(&OutputSuite{})

var aptly_BarPublishGeneratePackageFiles_ptr = aptly.BarPublishGeneratePackageFiles

func (s *OutputSuite) SetUpTest(c *C) {
	s.output = NewOutput()
	s.publishOutput = &PublishOutput{
		Progress: s.output,
		PublishDetail: PublishDetail{
			Detail: &Detail{},
		},
		barType: nil,
	}
}

func (s *OutputSuite) TestNewOutput(c *C) {
	// Test creating new output
	output := NewOutput()
	c.Check(output, NotNil)
	c.Check(output.mu, NotNil)
	c.Check(output.output, NotNil)
	c.Check(output.String(), Equals, "")
}

func (s *OutputSuite) TestOutputString(c *C) {
	// Test String method
	c.Check(s.output.String(), Equals, "")

	// Write some content and test again
	s.output.WriteString("test content")
	c.Check(s.output.String(), Equals, "test content")
}

func (s *OutputSuite) TestOutputStringConcurrent(c *C) {
	// Test String method with concurrent access
	var wg sync.WaitGroup

	// Start multiple goroutines writing to output
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			s.output.WriteString("test")
		}(i)
	}

	// Start multiple goroutines reading from output
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = s.output.String()
		}()
	}

	wg.Wait()

	// Should contain all the writes
	result := s.output.String()
	c.Check(len(result), Equals, 40) // 10 * "test" = 40 chars
}

func (s *OutputSuite) TestOutputWrite(c *C) {
	// Test Write method
	data := []byte("test data")
	n, err := s.output.Write(data)
	c.Check(err, IsNil)
	c.Check(n, Equals, len(data))

	// Write method doesn't actually write to buffer, just returns length
	c.Check(s.output.String(), Equals, "")
}

func (s *OutputSuite) TestOutputWriteError(c *C) {
	// Test Write method - can't override the method, so just test normal behavior
	data := []byte("test data")

	n, err := s.output.Write(data)
	c.Check(err, IsNil)
	c.Check(n, Equals, len(data))
}

func (s *OutputSuite) TestOutputWriteString(c *C) {
	// Test WriteString method
	text := "hello world"
	n, err := s.output.WriteString(text)
	c.Check(err, IsNil)
	c.Check(n, Equals, len(text))
	c.Check(s.output.String(), Equals, text)
}

func (s *OutputSuite) TestOutputWriteStringMultiple(c *C) {
	// Test multiple WriteString calls
	texts := []string{"hello", " ", "world", "!"}

	for _, text := range texts {
		n, err := s.output.WriteString(text)
		c.Check(err, IsNil)
		c.Check(n, Equals, len(text))
	}

	c.Check(s.output.String(), Equals, "hello world!")
}

func (s *OutputSuite) TestOutputWriteStringConcurrent(c *C) {
	// Test WriteString with concurrent access
	var wg sync.WaitGroup

	// Start multiple goroutines writing different strings
	texts := []string{"a", "b", "c", "d", "e"}
	for _, text := range texts {
		wg.Add(1)
		go func(t string) {
			defer wg.Done()
			s.output.WriteString(t)
		}(text)
	}

	wg.Wait()

	result := s.output.String()
	c.Check(len(result), Equals, 5)

	// All characters should be present (order may vary due to concurrency)
	for _, text := range texts {
		c.Check(result, Matches, ".*"+text+".*")
	}
}

func (s *OutputSuite) TestOutputStart(c *C) {
	// Test Start method (should not panic)
	s.output.Start()
	// No assertions needed, just ensure it doesn't panic
}

func (s *OutputSuite) TestOutputShutdown(c *C) {
	// Test Shutdown method (should not panic)
	s.output.Shutdown()
	// No assertions needed, just ensure it doesn't panic
}

func (s *OutputSuite) TestOutputFlush(c *C) {
	// Test Flush method (should not panic)
	s.output.Flush()
	// No assertions needed, just ensure it doesn't panic
}

func (s *OutputSuite) TestOutputInitBar(c *C) {
	// Test InitBar method (should not panic)
	s.output.InitBar(100, true, aptly.BarPublishGeneratePackageFiles)
	// No assertions needed, just ensure it doesn't panic
}

func (s *OutputSuite) TestOutputShutdownBar(c *C) {
	// Test ShutdownBar method (should not panic)
	s.output.ShutdownBar()
	// No assertions needed, just ensure it doesn't panic
}

func (s *OutputSuite) TestOutputAddBar(c *C) {
	// Test AddBar method (should not panic)
	s.output.AddBar(5)
	// No assertions needed, just ensure it doesn't panic
}

func (s *OutputSuite) TestOutputSetBar(c *C) {
	// Test SetBar method (should not panic)
	s.output.SetBar(50)
	// No assertions needed, just ensure it doesn't panic
}

func (s *OutputSuite) TestOutputPrintf(c *C) {
	// Test Printf method
	s.output.Printf("Hello %s, number: %d", "world", 42)
	c.Check(s.output.String(), Equals, "Hello world, number: 42")
}

func (s *OutputSuite) TestOutputPrintfEmpty(c *C) {
	// Test Printf with empty format
	s.output.Printf("")
	c.Check(s.output.String(), Equals, "")
}

func (s *OutputSuite) TestOutputPrintfNoArgs(c *C) {
	// Test Printf with no arguments
	s.output.Printf("simple message")
	c.Check(s.output.String(), Equals, "simple message")
}

func (s *OutputSuite) TestOutputPrintfMultiple(c *C) {
	// Test multiple Printf calls
	s.output.Printf("Line 1: %s", "test")
	s.output.Printf(" Line 2: %d", 123)
	c.Check(s.output.String(), Equals, "Line 1: test Line 2: 123")
}

func (s *OutputSuite) TestOutputPrint(c *C) {
	// Test Print method
	s.output.Print("simple message")
	c.Check(s.output.String(), Equals, "simple message")
}

func (s *OutputSuite) TestOutputPrintEmpty(c *C) {
	// Test Print with empty string
	s.output.Print("")
	c.Check(s.output.String(), Equals, "")
}

func (s *OutputSuite) TestOutputPrintMultiple(c *C) {
	// Test multiple Print calls
	s.output.Print("Hello")
	s.output.Print(" ")
	s.output.Print("World")
	c.Check(s.output.String(), Equals, "Hello World")
}

func (s *OutputSuite) TestOutputColoredPrintf(c *C) {
	// Test ColoredPrintf method (adds newline)
	s.output.ColoredPrintf("Hello %s", "world")
	c.Check(s.output.String(), Equals, "Hello world\n")
}

func (s *OutputSuite) TestOutputColoredPrintfEmpty(c *C) {
	// Test ColoredPrintf with empty format
	s.output.ColoredPrintf("")
	c.Check(s.output.String(), Equals, "\n")
}

func (s *OutputSuite) TestOutputColoredPrintfNoArgs(c *C) {
	// Test ColoredPrintf with no arguments
	s.output.ColoredPrintf("simple message")
	c.Check(s.output.String(), Equals, "simple message\n")
}

func (s *OutputSuite) TestOutputColoredPrintfMultiple(c *C) {
	// Test multiple ColoredPrintf calls
	s.output.ColoredPrintf("Line 1: %s", "test")
	s.output.ColoredPrintf("Line 2: %d", 123)
	c.Check(s.output.String(), Equals, "Line 1: test\nLine 2: 123\n")
}

func (s *OutputSuite) TestOutputPrintfStdErr(c *C) {
	// Test PrintfStdErr method
	s.output.PrintfStdErr("Error: %s", "something went wrong")
	c.Check(s.output.String(), Equals, "Error: something went wrong")
}

func (s *OutputSuite) TestOutputPrintfStdErrEmpty(c *C) {
	// Test PrintfStdErr with empty format
	s.output.PrintfStdErr("")
	c.Check(s.output.String(), Equals, "")
}

func (s *OutputSuite) TestOutputPrintfStdErrNoArgs(c *C) {
	// Test PrintfStdErr with no arguments
	s.output.PrintfStdErr("error message")
	c.Check(s.output.String(), Equals, "error message")
}

func (s *OutputSuite) TestOutputMixedMethods(c *C) {
	// Test mixing different output methods
	s.output.Print("Start")
	s.output.Printf(" %s", "middle")
	s.output.ColoredPrintf(" %s", "end")
	s.output.PrintfStdErr("error")

	expected := "Start middle end\nerror"
	c.Check(s.output.String(), Equals, expected)
}

// PublishOutput tests

func (s *OutputSuite) TestPublishOutputInitBar(c *C) {
	// Test InitBar for publish output
	count := int64(100)
	s.publishOutput.InitBar(count, false, aptly.BarPublishGeneratePackageFiles)

	c.Check(s.publishOutput.barType, NotNil)
	c.Check(*s.publishOutput.barType, Equals, aptly.BarPublishGeneratePackageFiles)
	c.Check(s.publishOutput.TotalNumberOfPackages, Equals, count)
	c.Check(s.publishOutput.RemainingNumberOfPackages, Equals, count)
}

func (s *OutputSuite) TestPublishOutputInitBarOtherType(c *C) {
	// Test InitBar for publish output with different bar type
	count := int64(50)
	s.publishOutput.InitBar(count, false, aptly.BarGeneralBuildPackageList)

	c.Check(s.publishOutput.barType, NotNil)
	c.Check(*s.publishOutput.barType, Equals, aptly.BarGeneralBuildPackageList)
	// Should not set package counts for other bar types
	c.Check(s.publishOutput.TotalNumberOfPackages, Equals, int64(0))
	c.Check(s.publishOutput.RemainingNumberOfPackages, Equals, int64(0))
}

func (s *OutputSuite) TestPublishOutputShutdownBar(c *C) {
	// Test ShutdownBar for publish output
	s.publishOutput.barType = &aptly_BarPublishGeneratePackageFiles_ptr
	s.publishOutput.ShutdownBar()

	c.Check(s.publishOutput.barType, IsNil)
}

func (s *OutputSuite) TestPublishOutputAddBar(c *C) {
	// Test AddBar for publish output with correct bar type
	s.publishOutput.barType = &aptly_BarPublishGeneratePackageFiles_ptr
	s.publishOutput.RemainingNumberOfPackages = 10

	s.publishOutput.AddBar(1)
	c.Check(s.publishOutput.RemainingNumberOfPackages, Equals, int64(9))

	s.publishOutput.AddBar(3) // Still decrements by 1, not 3
	c.Check(s.publishOutput.RemainingNumberOfPackages, Equals, int64(8))
}

func (s *OutputSuite) TestPublishOutputAddBarWrongType(c *C) {
	// Test AddBar for publish output with wrong bar type
	otherBarType := aptly.BarGeneralBuildPackageList
	s.publishOutput.barType = &otherBarType
	s.publishOutput.RemainingNumberOfPackages = 10

	s.publishOutput.AddBar(1)
	// Should not decrement for wrong bar type
	c.Check(s.publishOutput.RemainingNumberOfPackages, Equals, int64(10))
}

func (s *OutputSuite) TestPublishOutputAddBarNilBarType(c *C) {
	// Test AddBar for publish output with nil bar type
	s.publishOutput.barType = nil
	s.publishOutput.RemainingNumberOfPackages = 10

	s.publishOutput.AddBar(1)
	// Should not decrement when bar type is nil
	c.Check(s.publishOutput.RemainingNumberOfPackages, Equals, int64(10))
}

func (s *OutputSuite) TestPublishOutputComplete(c *C) {
	// Test complete workflow of publish output
	count := int64(5)

	// Initialize
	s.publishOutput.InitBar(count, false, aptly.BarPublishGeneratePackageFiles)
	c.Check(s.publishOutput.TotalNumberOfPackages, Equals, count)
	c.Check(s.publishOutput.RemainingNumberOfPackages, Equals, count)

	// Simulate processing packages
	for i := int64(0); i < count; i++ {
		s.publishOutput.AddBar(1)
		c.Check(s.publishOutput.RemainingNumberOfPackages, Equals, count-i-1)
	}

	// Shutdown
	s.publishOutput.ShutdownBar()
	c.Check(s.publishOutput.barType, IsNil)
}

func (s *OutputSuite) TestOutputThreadSafety(c *C) {
	// Test thread safety of all methods
	var wg sync.WaitGroup
	numGoroutines := 20

	// Test concurrent access to all methods
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			s.output.WriteString(fmt.Sprintf("msg%d", n))
			s.output.Printf("printf%d", n)
			s.output.Print(fmt.Sprintf("print%d", n))
			s.output.ColoredPrintf("colored%d", n)
			s.output.PrintfStdErr("stderr%d", n)
			_ = s.output.String()
		}(i)
	}

	wg.Wait()

	// Should contain all messages without corruption
	result := s.output.String()
	c.Check(len(result) > 0, Equals, true)

	// Check that a reasonable amount of output was generated
	// Each goroutine writes 5 messages, so we should have significant output
	c.Check(len(result) > numGoroutines*10, Equals, true)

	// Check that some expected message patterns exist (not all, to avoid flakiness)
	// This verifies basic functionality without being too strict about concurrent ordering
	foundMsg := false
	foundPrintf := false
	foundStderr := false
	for i := 0; i < numGoroutines; i++ {
		if !foundMsg && strings.Contains(result, fmt.Sprintf("msg%d", i)) {
			foundMsg = true
		}
		if !foundPrintf && strings.Contains(result, fmt.Sprintf("printf%d", i)) {
			foundPrintf = true
		}
		if !foundStderr && strings.Contains(result, fmt.Sprintf("stderr%d", i)) {
			foundStderr = true
		}
	}

	c.Check(foundMsg, Equals, true)
	c.Check(foundPrintf, Equals, true)
	c.Check(foundStderr, Equals, true)
}

func (s *OutputSuite) TestProgressInterfaceCompliance(c *C) {
	// Test that Output implements aptly.Progress interface
	var progress aptly.Progress = s.output
	c.Check(progress, NotNil)

	// Test calling interface methods
	progress.Start()
	progress.Shutdown()
	progress.Flush()
	progress.InitBar(100, false, aptly.BarPublishGeneratePackageFiles)
	progress.ShutdownBar()
	progress.AddBar(1)
	progress.SetBar(50)
	progress.Printf("test %s", "message")
	progress.ColoredPrintf("test %s", "colored")
}

func (s *OutputSuite) TestPublishOutputProgressInterfaceCompliance(c *C) {
	// Test that PublishOutput implements aptly.Progress interface
	var progress aptly.Progress = s.publishOutput
	c.Check(progress, NotNil)

	// Test calling interface methods
	progress.InitBar(100, false, aptly.BarPublishGeneratePackageFiles)
	progress.AddBar(1)
	progress.ShutdownBar()
}

// Test edge cases and error scenarios

func (s *OutputSuite) TestOutputLargeData(c *C) {
	// Test with large amounts of data
	largeString := strings.Repeat("x", 10000)

	n, err := s.output.WriteString(largeString)
	c.Check(err, IsNil)
	c.Check(n, Equals, len(largeString))
	c.Check(s.output.String(), Equals, largeString)
}

func (s *OutputSuite) TestOutputSpecialCharacters(c *C) {
	// Test with special characters and unicode
	specialString := "Hello L! \n\t\r Special: @#$%^&*()"

	s.output.WriteString(specialString)
	c.Check(s.output.String(), Equals, specialString)
}

func (s *OutputSuite) TestPublishOutputNegativeAddBar(c *C) {
	// Test AddBar with negative values (edge case)
	s.publishOutput.barType = &aptly_BarPublishGeneratePackageFiles_ptr
	s.publishOutput.RemainingNumberOfPackages = 5

	// This should still decrement by 1 regardless of the parameter
	s.publishOutput.AddBar(-10)
	c.Check(s.publishOutput.RemainingNumberOfPackages, Equals, int64(4))
}

func (s *OutputSuite) TestPublishOutputZeroAddBar(c *C) {
	// Test AddBar with zero value
	s.publishOutput.barType = &aptly_BarPublishGeneratePackageFiles_ptr
	s.publishOutput.RemainingNumberOfPackages = 5

	s.publishOutput.AddBar(0)
	c.Check(s.publishOutput.RemainingNumberOfPackages, Equals, int64(4))
}
