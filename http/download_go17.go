// +build go1.7

package http

import (
	"net/http"
)

func initTransport(transport *http.Transport) {
	transport.DialContext = http.DefaultTransport.(*http.Transport).DialContext
	transport.MaxIdleConns = http.DefaultTransport.(*http.Transport).MaxIdleConns
	transport.IdleConnTimeout = http.DefaultTransport.(*http.Transport).IdleConnTimeout
}
