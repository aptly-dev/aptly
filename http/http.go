// Package http provides all HTTP (and FTP)-related operations
package http

import (
	"fmt"
	"net/url"
)

// Error is download error connected to HTTP code
type Error struct {
	Code int
	URL  string
}

// Error
func (e *Error) Error() string {
	return fmt.Sprintf("HTTP code %d while fetching %s", e.Code, e.URL)
}

// NoCandidateFoundError indicates that now candidate of given url could be found
type NoCandidateFoundError struct {
	URL *url.URL
}

// Error message
func (e *NoCandidateFoundError) Error() string {
	return fmt.Sprintf("no candidates for %s found", e.URL)
}
