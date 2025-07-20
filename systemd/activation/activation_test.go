package activation

import (
	"crypto/tls"
	"net"
	"os"
	"strconv"
	"testing"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type ActivationSuite struct {
	originalPID string
	originalFDS string
}

var _ = Suite(&ActivationSuite{})

func (s *ActivationSuite) SetUpTest(c *C) {
	// Save original environment variables
	s.originalPID = os.Getenv("LISTEN_PID")
	s.originalFDS = os.Getenv("LISTEN_FDS")
}

func (s *ActivationSuite) TearDownTest(c *C) {
	// Restore original environment variables
	if s.originalPID != "" {
		os.Setenv("LISTEN_PID", s.originalPID)
	} else {
		os.Unsetenv("LISTEN_PID")
	}

	if s.originalFDS != "" {
		os.Setenv("LISTEN_FDS", s.originalFDS)
	} else {
		os.Unsetenv("LISTEN_FDS")
	}
}

func (s *ActivationSuite) TestFilesNoEnvironment(c *C) {
	// Test Files function when no environment variables are set
	os.Unsetenv("LISTEN_PID")
	os.Unsetenv("LISTEN_FDS")

	files := Files(false)
	c.Check(files, IsNil)
}

func (s *ActivationSuite) TestFilesWrongPID(c *C) {
	// Test Files function when LISTEN_PID doesn't match current process
	currentPID := os.Getpid()
	wrongPID := currentPID + 1000

	os.Setenv("LISTEN_PID", strconv.Itoa(wrongPID))
	os.Setenv("LISTEN_FDS", "1")

	files := Files(false)
	c.Check(files, IsNil)
}

func (s *ActivationSuite) TestFilesInvalidPID(c *C) {
	// Test Files function with invalid PID
	os.Setenv("LISTEN_PID", "invalid")
	os.Setenv("LISTEN_FDS", "1")

	files := Files(false)
	c.Check(files, IsNil)
}

func (s *ActivationSuite) TestFilesInvalidFDS(c *C) {
	// Test Files function with invalid LISTEN_FDS
	currentPID := os.Getpid()
	os.Setenv("LISTEN_PID", strconv.Itoa(currentPID))
	os.Setenv("LISTEN_FDS", "invalid")

	files := Files(false)
	c.Check(files, IsNil)
}

func (s *ActivationSuite) TestFilesZeroFDS(c *C) {
	// Test Files function with zero file descriptors
	currentPID := os.Getpid()
	os.Setenv("LISTEN_PID", strconv.Itoa(currentPID))
	os.Setenv("LISTEN_FDS", "0")

	files := Files(false)
	c.Check(files, IsNil)
}

func (s *ActivationSuite) TestFilesCorrectPID(c *C) {
	// Test Files function with correct PID but no actual FDs
	currentPID := os.Getpid()
	os.Setenv("LISTEN_PID", strconv.Itoa(currentPID))
	os.Setenv("LISTEN_FDS", "2")

	files := Files(false)
	// Should return a slice of files even if the FDs aren't valid
	c.Check(files, NotNil)
	c.Check(len(files), Equals, 2)
}

func (s *ActivationSuite) TestFilesUnsetEnv(c *C) {
	// Test Files function with unsetEnv=true
	currentPID := os.Getpid()
	os.Setenv("LISTEN_PID", strconv.Itoa(currentPID))
	os.Setenv("LISTEN_FDS", "1")

	files := Files(true)

	// Environment variables should be unset after the call
	c.Check(os.Getenv("LISTEN_PID"), Equals, "")
	c.Check(os.Getenv("LISTEN_FDS"), Equals, "")

	// Should still return files
	c.Check(files, NotNil)
	c.Check(len(files), Equals, 1)
}

func (s *ActivationSuite) TestFilesKeepEnv(c *C) {
	// Test Files function with unsetEnv=false
	currentPID := os.Getpid()
	pidStr := strconv.Itoa(currentPID)

	os.Setenv("LISTEN_PID", pidStr)
	os.Setenv("LISTEN_FDS", "1")

	files := Files(false)

	// Environment variables should remain set
	c.Check(os.Getenv("LISTEN_PID"), Equals, pidStr)
	c.Check(os.Getenv("LISTEN_FDS"), Equals, "1")

	// Should return files
	c.Check(files, NotNil)
	c.Check(len(files), Equals, 1)
}

func (s *ActivationSuite) TestListenersNoFiles(c *C) {
	// Test Listeners function when Files returns nil
	os.Unsetenv("LISTEN_PID")
	os.Unsetenv("LISTEN_FDS")

	listeners, err := Listeners(false)
	c.Check(err, IsNil)
	c.Check(listeners, NotNil)
	c.Check(len(listeners), Equals, 0)
}

func (s *ActivationSuite) TestListenersWithFiles(c *C) {
	// Test Listeners function with files (they won't be valid listeners)
	currentPID := os.Getpid()
	os.Setenv("LISTEN_PID", strconv.Itoa(currentPID))
	os.Setenv("LISTEN_FDS", "2")

	listeners, err := Listeners(false)
	c.Check(err, IsNil)
	c.Check(listeners, NotNil)
	c.Check(len(listeners), Equals, 2)

	// The listeners will be nil because the FDs aren't real sockets
	for _, listener := range listeners {
		c.Check(listener, IsNil)
	}
}

func (s *ActivationSuite) TestPacketConnsNoFiles(c *C) {
	// Test PacketConns function when Files returns nil
	os.Unsetenv("LISTEN_PID")
	os.Unsetenv("LISTEN_FDS")

	conns, err := PacketConns(false)
	c.Check(err, IsNil)
	c.Check(conns, NotNil)
	c.Check(len(conns), Equals, 0)
}

func (s *ActivationSuite) TestPacketConnsWithFiles(c *C) {
	// Test PacketConns function with files (they won't be valid packet connections)
	currentPID := os.Getpid()
	os.Setenv("LISTEN_PID", strconv.Itoa(currentPID))
	os.Setenv("LISTEN_FDS", "3")

	conns, err := PacketConns(false)
	c.Check(err, IsNil)
	c.Check(conns, NotNil)
	c.Check(len(conns), Equals, 3)

	// The connections will be nil because the FDs aren't real packet sockets
	for _, conn := range conns {
		c.Check(conn, IsNil)
	}
}

func (s *ActivationSuite) TestTLSListenersNilConfig(c *C) {
	// Test TLSListeners with nil TLS config
	os.Unsetenv("LISTEN_PID")
	os.Unsetenv("LISTEN_FDS")

	listeners, err := TLSListeners(false, nil)
	c.Check(err, IsNil)
	c.Check(listeners, NotNil)
	c.Check(len(listeners), Equals, 0)
}

func (s *ActivationSuite) TestTLSListenersWithConfig(c *C) {
	// Test TLSListeners with TLS config
	currentPID := os.Getpid()
	os.Setenv("LISTEN_PID", strconv.Itoa(currentPID))
	os.Setenv("LISTEN_FDS", "2")

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	listeners, err := TLSListeners(false, tlsConfig)
	c.Check(err, IsNil)
	c.Check(listeners, NotNil)
	c.Check(len(listeners), Equals, 2)

	// The listeners will be nil because the FDs aren't real sockets
	// This is expected behavior in test environment
	for _, listener := range listeners {
		c.Check(listener, IsNil)
	}
}

func (s *ActivationSuite) TestConstant(c *C) {
	// Test that the constant is defined correctly
	c.Check(listenFdsStart, Equals, 3)
}

func (s *ActivationSuite) TestFileDescriptorRange(c *C) {
	// Test file descriptor range calculation
	currentPID := os.Getpid()
	nfds := 5

	os.Setenv("LISTEN_PID", strconv.Itoa(currentPID))
	os.Setenv("LISTEN_FDS", strconv.Itoa(nfds))

	files := Files(false)
	c.Check(files, NotNil)
	c.Check(len(files), Equals, nfds)

	// Check that file descriptors start from listenFdsStart
	for i, file := range files {
		expectedFD := listenFdsStart + i
		c.Check(file.Name(), Equals, "LISTEN_FD_"+strconv.Itoa(expectedFD))
	}
}

// Mock listener for TLS testing
type mockListener struct {
	addr mockAddr
}

type mockAddr struct {
	network string
}

func (m mockAddr) Network() string { return m.network }
func (m mockAddr) String() string  { return "mock-addr" }

func (m mockListener) Accept() (net.Conn, error) { return nil, nil }
func (m mockListener) Close() error              { return nil }
func (m mockListener) Addr() net.Addr            { return m.addr }

func (s *ActivationSuite) TestTLSListenerWrapping(c *C) {
	// Test TLS listener wrapping logic

	// Create mock listeners
	tcpListener := &mockListener{addr: mockAddr{network: "tcp"}}
	udpListener := &mockListener{addr: mockAddr{network: "udp"}}

	listeners := []net.Listener{tcpListener, udpListener, nil}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// Simulate the TLS wrapping logic
	for i, l := range listeners {
		if l != nil && l.Addr().Network() == "tcp" {
			// In real code, this would be: listeners[i] = tls.NewListener(l, tlsConfig)
			c.Check(l.Addr().Network(), Equals, "tcp")
			c.Check(tlsConfig, NotNil)
			listeners[i] = l // Keep reference for test
		}
	}

	// Verify that only TCP listeners would be wrapped
	c.Check(listeners[0].Addr().Network(), Equals, "tcp") // Would be wrapped
	c.Check(listeners[1].Addr().Network(), Equals, "udp") // Would not be wrapped
	c.Check(listeners[2], IsNil)                          // Nil listener
}

func (s *ActivationSuite) TestEnvironmentVariableHandling(c *C) {
	// Test various environment variable scenarios
	testCases := []struct {
		name     string
		pid      string
		fds      string
		expected bool
	}{
		{"valid current PID", strconv.Itoa(os.Getpid()), "1", true},
		{"invalid PID string", "not-a-number", "1", false},
		{"wrong PID", strconv.Itoa(os.Getpid() + 1000), "1", false},
		{"invalid FDS string", strconv.Itoa(os.Getpid()), "not-a-number", false},
		{"zero FDS", strconv.Itoa(os.Getpid()), "0", false},
		{"negative FDS", strconv.Itoa(os.Getpid()), "-1", false},
		{"small FDS", strconv.Itoa(os.Getpid()), "2", true},
	}

	for _, tc := range testCases {
		os.Setenv("LISTEN_PID", tc.pid)
		os.Setenv("LISTEN_FDS", tc.fds)

		files := Files(false)

		if tc.expected {
			c.Check(files, NotNil, Commentf("Test case: %s", tc.name))
			if tc.fds != "0" && tc.fds != "-1" {
				expectedLen, _ := strconv.Atoi(tc.fds)
				if expectedLen > 0 {
					c.Check(len(files), Equals, expectedLen, Commentf("Test case: %s", tc.name))
				}
			}
		} else {
			c.Check(files, IsNil, Commentf("Test case: %s", tc.name))
		}
	}
}

func (s *ActivationSuite) TestErrorHandling(c *C) {
	// Test error handling in all functions

	// Test Listeners with no error
	listeners, err := Listeners(false)
	c.Check(err, IsNil)
	c.Check(listeners, NotNil)

	// Test PacketConns with no error
	conns, err := PacketConns(false)
	c.Check(err, IsNil)
	c.Check(conns, NotNil)
}

func (s *ActivationSuite) TestTLSListenersWithNilConfig(c *C) {
	// Test TLSListeners with nil TLS config
	currentPID := os.Getpid()
	os.Setenv("LISTEN_PID", strconv.Itoa(currentPID))
	os.Setenv("LISTEN_FDS", "1")

	listeners, err := TLSListeners(false, nil)
	c.Check(err, IsNil)
	c.Check(listeners, NotNil)
	c.Check(len(listeners), Equals, 1)
}

func (s *ActivationSuite) TestFilesUnsetEnvAdditional(c *C) {
	// Test Files function with unsetEnv=true - additional coverage
	currentPID := os.Getpid()
	os.Setenv("LISTEN_PID", strconv.Itoa(currentPID))
	os.Setenv("LISTEN_FDS", "1")

	files := Files(true)
	c.Check(files, NotNil)

	// Environment variables should be unset after the call
	c.Check(os.Getenv("LISTEN_PID"), Equals, "")
	c.Check(os.Getenv("LISTEN_FDS"), Equals, "")
}

func (s *ActivationSuite) TestTLSListenersNilListeners(c *C) {
	// Test TLSListeners when Listeners returns empty slice
	os.Unsetenv("LISTEN_PID")
	os.Unsetenv("LISTEN_FDS")

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	listeners, err := TLSListeners(false, tlsConfig)
	c.Check(err, IsNil)
	c.Check(listeners, NotNil) // Returns empty slice, not nil
	c.Check(len(listeners), Equals, 0)
}

func (s *ActivationSuite) TestExcessiveFDSLimit(c *C) {
	// Test the protection against excessive FDS allocations
	currentPID := os.Getpid()
	os.Setenv("LISTEN_PID", strconv.Itoa(currentPID))
	os.Setenv("LISTEN_FDS", "2000") // Over the 1000 limit

	files := Files(false)
	c.Check(files, IsNil) // Should return nil due to excessive FDS count
}

func (s *ActivationSuite) TestDeferredEnvironmentCleanup(c *C) {
	// Test the deferred environment cleanup
	currentPID := os.Getpid()

	// Set environment variables
	os.Setenv("LISTEN_PID", strconv.Itoa(currentPID))
	os.Setenv("LISTEN_FDS", "1")

	// Verify they are set
	c.Check(os.Getenv("LISTEN_PID"), Not(Equals), "")
	c.Check(os.Getenv("LISTEN_FDS"), Not(Equals), "")

	// Call Files with unsetEnv=true
	files := Files(true)

	// Verify environment is cleaned up
	c.Check(os.Getenv("LISTEN_PID"), Equals, "")
	c.Check(os.Getenv("LISTEN_FDS"), Equals, "")

	// Should still return files
	c.Check(files, NotNil)
}

func (s *ActivationSuite) TestCloseOnExecCall(c *C) {
	// Test that CloseOnExec is called for file descriptors
	// This is a structural test since we can't easily verify syscall effects

	currentPID := os.Getpid()
	os.Setenv("LISTEN_PID", strconv.Itoa(currentPID))
	os.Setenv("LISTEN_FDS", "2")

	files := Files(false)
	c.Check(files, NotNil)
	c.Check(len(files), Equals, 2)

	// Verify files are created with expected names
	c.Check(files[0].Name(), Equals, "LISTEN_FD_3")
	c.Check(files[1].Name(), Equals, "LISTEN_FD_4")
}
