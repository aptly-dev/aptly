package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/deb"
	"github.com/smira/commander"
	"github.com/smira/flag"
	. "gopkg.in/check.v1"
)

type SnapshotListSuite struct {
	cmd               *commander.Command
	collectionFactory *deb.CollectionFactory
	mockProgress      *MockSnapshotListProgress
	mockContext       *MockSnapshotListContext
}

var _ = Suite(&SnapshotListSuite{})

func (s *SnapshotListSuite) SetUpTest(c *C) {
	s.cmd = makeCmdSnapshotList()
	s.mockProgress = &MockSnapshotListProgress{}

	// Set up mock collections
	s.collectionFactory = &deb.CollectionFactory{
		snapshotCollection: &MockSnapshotListCollection{},
	}

	// Set up mock context
	s.mockContext = &MockSnapshotListContext{
		flags:             s.cmd.Flag,
		progress:          s.mockProgress,
		collectionFactory: s.collectionFactory,
	}

	// Set up required flags
	s.cmd.Flag.Bool("json", false, "display list in JSON format")
	s.cmd.Flag.Bool("raw", false, "display list in machine-readable format")
	s.cmd.Flag.String("sort", "name", "sort method")

	// Set mock context globally
	context = s.mockContext
}

func (s *SnapshotListSuite) TestMakeCmdSnapshotList(c *C) {
	// Test command creation and basic properties
	cmd := makeCmdSnapshotList()
	c.Check(cmd, NotNil)
	c.Check(cmd.UsageLine, Equals, "list")
	c.Check(cmd.Short, Equals, "list snapshots")
	c.Check(strings.Contains(cmd.Long, "Command list shows full list of snapshots created"), Equals, true)

	// Test flags
	jsonFlag := cmd.Flag.Lookup("json")
	c.Check(jsonFlag, NotNil)
	c.Check(jsonFlag.DefValue, Equals, "false")

	rawFlag := cmd.Flag.Lookup("raw")
	c.Check(rawFlag, NotNil)
	c.Check(rawFlag.DefValue, Equals, "false")

	sortFlag := cmd.Flag.Lookup("sort")
	c.Check(sortFlag, NotNil)
	c.Check(sortFlag.DefValue, Equals, "name")
}

func (s *SnapshotListSuite) TestAptlySnapshotListInvalidArgs(c *C) {
	// Test with arguments (should not accept any)
	err := aptlySnapshotList(s.cmd, []string{"invalid", "args"})
	c.Check(err, Equals, commander.ErrCommandError)
}

func (s *SnapshotListSuite) TestAptlySnapshotListTxtBasic(c *C) {
	// Test basic text output
	args := []string{}

	// Capture stdout since the function prints directly
	var output strings.Builder
	originalPrintf := fmt.Printf
	defer func() { fmt.Printf = originalPrintf }()

	fmt.Printf = func(format string, a ...interface{}) (n int, err error) {
		return output.WriteString(fmt.Sprintf(format, a...))
	}

	err := aptlySnapshotList(s.cmd, args)
	c.Check(err, IsNil)

	// Check output contains expected content
	outputStr := output.String()
	c.Check(strings.Contains(outputStr, "List of snapshots:"), Equals, true)
	c.Check(strings.Contains(outputStr, "test-snapshot"), Equals, true)
	c.Check(strings.Contains(outputStr, "aptly snapshot show"), Equals, true)
}

func (s *SnapshotListSuite) TestAptlySnapshotListTxtEmpty(c *C) {
	// Test with no snapshots
	mockCollection := &MockSnapshotListCollection{emptyCollection: true}
	s.collectionFactory.snapshotCollection = mockCollection

	var output strings.Builder
	originalPrintf := fmt.Printf
	defer func() { fmt.Printf = originalPrintf }()

	fmt.Printf = func(format string, a ...interface{}) (n int, err error) {
		return output.WriteString(fmt.Sprintf(format, a...))
	}

	err := aptlySnapshotList(s.cmd, []string{})
	c.Check(err, IsNil)

	// Should display message about no snapshots
	outputStr := output.String()
	c.Check(strings.Contains(outputStr, "No snapshots found"), Equals, true)
	c.Check(strings.Contains(outputStr, "aptly snapshot create"), Equals, true)
}

func (s *SnapshotListSuite) TestAptlySnapshotListTxtRaw(c *C) {
	// Test raw output format
	s.cmd.Flag.Set("raw", "true")

	var output strings.Builder
	originalPrintf := fmt.Printf
	defer func() { fmt.Printf = originalPrintf }()

	fmt.Printf = func(format string, a ...interface{}) (n int, err error) {
		return output.WriteString(fmt.Sprintf(format, a...))
	}

	err := aptlySnapshotList(s.cmd, []string{})
	c.Check(err, IsNil)

	// Should display raw format (just snapshot names)
	outputStr := output.String()
	c.Check(strings.Contains(outputStr, "test-snapshot"), Equals, true)
	c.Check(strings.Contains(outputStr, "List of snapshots:"), Equals, false) // No header in raw mode
}

func (s *SnapshotListSuite) TestAptlySnapshotListJSON(c *C) {
	// Test JSON output
	s.cmd.Flag.Set("json", "true")

	var output strings.Builder
	originalPrintln := fmt.Println
	defer func() { fmt.Println = originalPrintln }()

	fmt.Println = func(a ...interface{}) (n int, err error) {
		return output.WriteString(fmt.Sprintln(a...))
	}

	err := aptlySnapshotList(s.cmd, args)
	c.Check(err, IsNil)

	// Should output valid JSON
	outputStr := output.String()
	c.Check(strings.Contains(outputStr, "{"), Equals, true)
	c.Check(strings.Contains(outputStr, "}"), Equals, true)
	
	// Verify it's valid JSON
	var snapshots []interface{}
	err = json.Unmarshal([]byte(strings.TrimSpace(outputStr)), &snapshots)
	c.Check(err, IsNil)
}

func (s *SnapshotListSuite) TestAptlySnapshotListJSONEmpty(c *C) {
	// Test JSON output with empty collection
	s.cmd.Flag.Set("json", "true")
	mockCollection := &MockSnapshotListCollection{emptyCollection: true}
	s.collectionFactory.snapshotCollection = mockCollection

	var output strings.Builder
	originalPrintln := fmt.Println
	defer func() { fmt.Println = originalPrintln }()

	fmt.Println = func(a ...interface{}) (n int, err error) {
		return output.WriteString(fmt.Sprintln(a...))
	}

	err := aptlySnapshotList(s.cmd, []string{})
	c.Check(err, IsNil)

	// Should output empty JSON array
	outputStr := strings.TrimSpace(output.String())
	c.Check(strings.Contains(outputStr, "[]"), Equals, true)
}

func (s *SnapshotListSuite) TestAptlySnapshotListJSONMarshalError(c *C) {
	// Test JSON marshal error
	s.cmd.Flag.Set("json", "true")
	mockCollection := &MockSnapshotListCollection{causeMarshalError: true}
	s.collectionFactory.snapshotCollection = mockCollection

	err := aptlySnapshotList(s.cmd, []string{})
	c.Check(err, NotNil) // Should fail on marshal error
}

func (s *SnapshotListSuite) TestAptlySnapshotListSortByName(c *C) {
	// Test sorting by name
	s.cmd.Flag.Set("sort", "name")
	mockCollection := &MockSnapshotListCollection{
		multipleSnapshots: true,
		snapshotNames:     []string{"z-snapshot", "a-snapshot", "m-snapshot"},
	}
	s.collectionFactory.snapshotCollection = mockCollection

	var output strings.Builder
	originalPrintf := fmt.Printf
	defer func() { fmt.Printf = originalPrintf }()

	fmt.Printf = func(format string, a ...interface{}) (n int, err error) {
		return output.WriteString(fmt.Sprintf(format, a...))
	}

	err := aptlySnapshotList(s.cmd, []string{})
	c.Check(err, IsNil)

	// Should complete successfully with sorted output
	outputStr := output.String()
	c.Check(strings.Contains(outputStr, "List of snapshots:"), Equals, true)
}

func (s *SnapshotListSuite) TestAptlySnapshotListSortByTime(c *C) {
	// Test sorting by time
	s.cmd.Flag.Set("sort", "time")
	mockCollection := &MockSnapshotListCollection{
		multipleSnapshots: true,
		snapshotNames:     []string{"old-snapshot", "new-snapshot", "middle-snapshot"},
	}
	s.collectionFactory.snapshotCollection = mockCollection

	var output strings.Builder
	originalPrintf := fmt.Printf
	defer func() { fmt.Printf = originalPrintf }()

	fmt.Printf = func(format string, a ...interface{}) (n int, err error) {
		return output.WriteString(fmt.Sprintf(format, a...))
	}

	err := aptlySnapshotList(s.cmd, []string{})
	c.Check(err, IsNil)

	// Should complete successfully with time-sorted output
	outputStr := output.String()
	c.Check(strings.Contains(outputStr, "List of snapshots:"), Equals, true)
}

func (s *SnapshotListSuite) TestAptlySnapshotListForEachError(c *C) {
	// Test with error during snapshot iteration
	mockCollection := &MockSnapshotListCollection{shouldErrorForEach: true}
	s.collectionFactory.snapshotCollection = mockCollection

	err := aptlySnapshotList(s.cmd, []string{})
	c.Check(err, NotNil) // ForEach errors should be returned
}

func (s *SnapshotListSuite) TestAptlySnapshotListJSONSorting(c *C) {
	// Test JSON output with sorting
	s.cmd.Flag.Set("json", "true")
	s.cmd.Flag.Set("sort", "name")
	mockCollection := &MockSnapshotListCollection{
		multipleSnapshots: true,
		snapshotNames:     []string{"z-snapshot", "a-snapshot", "m-snapshot"},
	}
	s.collectionFactory.snapshotCollection = mockCollection

	var output strings.Builder
	originalPrintln := fmt.Println
	defer func() { fmt.Println = originalPrintln }()

	fmt.Println = func(a ...interface{}) (n int, err error) {
		return output.WriteString(fmt.Sprintln(a...))
	}

	err := aptlySnapshotList(s.cmd, []string{})
	c.Check(err, IsNil)

	// Should complete successfully with sorted JSON output
	outputStr := output.String()
	c.Check(len(outputStr) > 0, Equals, true)
	
	// Verify it's valid JSON
	var snapshots []map[string]interface{}
	err = json.Unmarshal([]byte(strings.TrimSpace(outputStr)), &snapshots)
	c.Check(err, IsNil)
	c.Check(len(snapshots), Equals, 3)
}

// Mock implementations for testing

type MockSnapshotListProgress struct {
	Messages []string
}

func (m *MockSnapshotListProgress) Printf(msg string, a ...interface{}) {
	formatted := fmt.Sprintf(msg, a...)
	m.Messages = append(m.Messages, formatted)
}

type MockSnapshotListContext struct {
	flags             *flag.FlagSet
	progress          *MockSnapshotListProgress
	collectionFactory *deb.CollectionFactory
}

func (m *MockSnapshotListContext) Flags() *flag.FlagSet                          { return m.flags }
func (m *MockSnapshotListContext) Progress() aptly.Progress                      { return m.progress }
func (m *MockSnapshotListContext) NewCollectionFactory() *deb.CollectionFactory { return m.collectionFactory }

type MockSnapshotListCollection struct {
	emptyCollection     bool
	shouldErrorForEach  bool
	causeMarshalError   bool
	multipleSnapshots   bool
	snapshotNames       []string
}

func (m *MockSnapshotListCollection) Len() int {
	if m.emptyCollection {
		return 0
	}
	if m.multipleSnapshots {
		return len(m.snapshotNames)
	}
	return 1
}

func (m *MockSnapshotListCollection) ForEachSorted(sortMethod string, handler func(*deb.Snapshot) error) error {
	if m.shouldErrorForEach {
		return fmt.Errorf("mock for each error")
	}

	if m.emptyCollection {
		return nil
	}

	if m.multipleSnapshots {
		// Sort snapshots based on method
		names := make([]string, len(m.snapshotNames))
		copy(names, m.snapshotNames)
		
		if sortMethod == "name" {
			// Sort alphabetically for name sorting
			for i := 0; i < len(names)-1; i++ {
				for j := i + 1; j < len(names); j++ {
					if names[i] > names[j] {
						names[i], names[j] = names[j], names[i]
					}
				}
			}
		}
		// For time sorting, keep original order (simulate time-based order)

		for _, name := range names {
			snapshot := &deb.Snapshot{
				Name:        name,
				Description: "Test snapshot",
			}
			
			if err := handler(snapshot); err != nil {
				return err
			}
		}
	} else {
		snapshot := &deb.Snapshot{
			Name:        "test-snapshot",
			Description: "Test snapshot",
		}
		
		// Create problematic snapshot for marshal error testing
		if m.causeMarshalError {
			// Create a cyclic structure that can't be marshaled
			snapshot.TestCyclicRef = snapshot
		}
		
		return handler(snapshot)
	}

	return nil
}

// Add methods to support snapshot operations
func (s *deb.Snapshot) String() string {
	return fmt.Sprintf("%s: %s", s.Name, s.Description)
}

// Test JSON marshaling directly
func (s *SnapshotListSuite) TestJSONMarshalDirect(c *C) {
	// Test JSON marshaling of snapshots directly
	snapshots := []*deb.Snapshot{
		{Name: "snapshot1", Description: "First snapshot"},
		{Name: "snapshot2", Description: "Second snapshot"},
	}

	output, err := json.MarshalIndent(snapshots, "", "  ")
	c.Check(err, IsNil)
	c.Check(len(output) > 0, Equals, true)
	c.Check(strings.Contains(string(output), "snapshot1"), Equals, true)
}

// Test sorting methods
func (s *SnapshotListSuite) TestSortingMethods(c *C) {
	// Test different sorting methods
	sortMethods := []string{"name", "time"}

	for _, method := range sortMethods {
		s.cmd.Flag.Set("sort", method)
		mockCollection := &MockSnapshotListCollection{
			multipleSnapshots: true,
			snapshotNames:     []string{"c-snapshot", "a-snapshot", "b-snapshot"},
		}
		s.collectionFactory.snapshotCollection = mockCollection

		var output strings.Builder
		originalPrintf := fmt.Printf
		fmt.Printf = func(format string, a ...interface{}) (n int, err error) {
			return output.WriteString(fmt.Sprintf(format, a...))
		}

		err := aptlySnapshotList(s.cmd, []string{})
		c.Check(err, IsNil, Commentf("Sort method: %s", method))

		outputStr := output.String()
		c.Check(strings.Contains(outputStr, "List of snapshots:"), Equals, true)

		fmt.Printf = originalPrintf
	}
}

// Test flag combinations
func (s *SnapshotListSuite) TestFlagCombinations(c *C) {
	// Test various flag combinations
	flagCombinations := []map[string]string{
		{"json": "true", "sort": "name"},
		{"json": "true", "sort": "time"},
		{"raw": "true", "sort": "name"},
		{"raw": "true", "sort": "time"},
	}

	for _, flags := range flagCombinations {
		// Set flags
		for flag, value := range flags {
			s.cmd.Flag.Set(flag, value)
		}

		var output strings.Builder
		originalPrintf := fmt.Printf
		originalPrintln := fmt.Println

		fmt.Printf = func(format string, a ...interface{}) (n int, err error) {
			return output.WriteString(fmt.Sprintf(format, a...))
		}
		fmt.Println = func(a ...interface{}) (n int, err error) {
			return output.WriteString(fmt.Sprintln(a...))
		}

		err := aptlySnapshotList(s.cmd, []string{})
		c.Check(err, IsNil, Commentf("Flag combination: %v", flags))

		// Reset flags
		for flag := range flags {
			if flag == "sort" {
				s.cmd.Flag.Set(flag, "name")
			} else {
				s.cmd.Flag.Set(flag, "false")
			}
		}
		
		fmt.Printf = originalPrintf
		fmt.Println = originalPrintln
	}
}

// Test different snapshot configurations
func (s *SnapshotListSuite) TestSnapshotConfigurations(c *C) {
	// Test different snapshot setups
	configurations := []struct {
		emptyCollection   bool
		multipleSnapshots bool
		snapshotCount     int
	}{
		{true, false, 0},
		{false, false, 1},
		{false, true, 3},
	}

	for _, config := range configurations {
		mockCollection := &MockSnapshotListCollection{
			emptyCollection:   config.emptyCollection,
			multipleSnapshots: config.multipleSnapshots,
			snapshotNames:     []string{"a-snapshot", "b-snapshot", "c-snapshot"},
		}
		s.collectionFactory.snapshotCollection = mockCollection

		var output strings.Builder
		originalPrintf := fmt.Printf
		fmt.Printf = func(format string, a ...interface{}) (n int, err error) {
			return output.WriteString(fmt.Sprintf(format, a...))
		}

		err := aptlySnapshotList(s.cmd, []string{})
		c.Check(err, IsNil, Commentf("Configuration: %+v", config))

		outputStr := output.String()
		if config.emptyCollection {
			c.Check(strings.Contains(outputStr, "No snapshots found"), Equals, true)
		} else {
			if config.multipleSnapshots {
				c.Check(strings.Contains(outputStr, "List of snapshots:"), Equals, true)
			} else {
				c.Check(strings.Contains(outputStr, "test-snapshot"), Equals, true)
			}
		}

		fmt.Printf = originalPrintf
	}
}

// Test edge cases
func (s *SnapshotListSuite) TestEdgeCases(c *C) {
	// Test with snapshot that has minimal configuration
	snapshot := &deb.Snapshot{Name: "simple-snapshot"}
	c.Check(snapshot.Name, Equals, "simple-snapshot")
	
	// Test string representation with minimal data
	stringRep := snapshot.String()
	c.Check(strings.Contains(stringRep, "simple-snapshot"), Equals, true)
}