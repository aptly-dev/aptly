// Package http provides all HTTP (and FTP)-related operations
package http

import (
	"fmt"
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
