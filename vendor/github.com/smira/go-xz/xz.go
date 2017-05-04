// Package xz implements simple .xz decompression using external xz program
//
// No shared library (liblzma) dependencies.
package xz

import (
	"io"
	"os/exec"
)

// Reader does decompression using xz utility
type Reader struct {
	cmd    *exec.Cmd
	input  io.WriteCloser
	output io.ReadCloser
}

// NewReader creates .xz decompression reader
//
// Internally it starts xz program, sets up input and output pipes
func NewReader(src io.Reader) (*Reader, error) {
	var err error

	result := &Reader{}

	result.cmd = exec.Command("xz", "--decompress", "--stdout")
	result.input, err = result.cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	result.output, err = result.cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	err = result.cmd.Start()
	if err != nil {
		return nil, err
	}

	go func() {
		io.Copy(result.input, src)
		result.input.Close()
	}()

	return result, nil
}

// Read implements io.Reader interface
func (r *Reader) Read(p []byte) (n int, err error) {
	return r.output.Read(p)
}

// Close implements io.Closer interface
func (r *Reader) Close() error {
	if r.input != nil {
		r.input.Close()
	}
	if r.output != nil {
		r.output.Close()
	}
	if r.cmd != nil {
		return r.cmd.Wait()
	}
	return nil
}

// Check interface
var (
	_ io.ReadCloser = &Reader{}
)
