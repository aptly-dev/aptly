// Package protocol implements  ftp:// scheme plugin for http.Transport
//
// github.com/jlaffaye/ftp library is used internally as FTP client implementation.
//
// Limitations: only anonymous FTP servers, only file retrieval operations.
//
// Internally connections to FTP servers are cached and re-used when possible.
//
// Example:
//
//    transport := &http.Transport{}
//    transport.RegisterProtocol("ftp", &FTPRoundTripper{})
//    client := &http.Client{Transport: transport}
//    resp, err := client.Get("ftp://ftp.ru.debian.org/debian/README")
package protocol

import (
	"fmt"
	"github.com/jlaffaye/ftp"
	"io"
	"net/http"
	"net/textproto"
	"strings"
	"sync"
)

// FTPRoundTripper is an implementation of net/http.RoundTripper on top of FTP client
type FTPRoundTripper struct {
	lock            sync.Mutex
	idleConnections map[string][]*ftp.ServerConn
}

// verify interface
var (
	_ http.RoundTripper = &FTPRoundTripper{}
)

type readCloserWrapper struct {
	body     io.ReadCloser
	rt       *FTPRoundTripper
	hostport string
	conn     *ftp.ServerConn
}

func (w *readCloserWrapper) Read(p []byte) (n int, err error) {
	return w.body.Read(p)
}

func (w *readCloserWrapper) Close() error {
	err := w.body.Close()
	if err == nil {
		w.rt.putConnection(w.hostport, w.conn)
	}

	return err
}

func (rt *FTPRoundTripper) getConnection(hostport string) (conn *ftp.ServerConn, err error) {
	rt.lock.Lock()
	conns, ok := rt.idleConnections[hostport]
	if ok && len(conns) > 0 {
		conn = conns[0]
		rt.idleConnections[hostport] = conns[1:]
		rt.lock.Unlock()
		return
	}
	rt.lock.Unlock()

	conn, err = ftp.Connect(hostport)
	if err != nil {
		return nil, err
	}

	err = conn.Login("anonymous", "anonymous")
	if err != nil {
		conn.Quit()
		return nil, err
	}

	return conn, nil
}

func (rt *FTPRoundTripper) putConnection(hostport string, conn *ftp.ServerConn) {
	rt.lock.Lock()
	defer rt.lock.Unlock()

	if rt.idleConnections == nil {
		rt.idleConnections = make(map[string][]*ftp.ServerConn)
	}

	rt.idleConnections[hostport] = append(rt.idleConnections[hostport], conn)
}

// RoundTrip parses incoming GET "HTTP" request and transforms it into
// commands to ftp client
func (rt *FTPRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.URL.Scheme != "ftp" {
		return nil, fmt.Errorf("only ftp protocol is supported, got %s", request.Proto)
	}

	if request.Method != "GET" {
		return nil, fmt.Errorf("only GET method is supported, got %s", request.Method)
	}

	hostport := request.URL.Host
	if strings.Index(hostport, ":") == -1 {
		hostport = hostport + ":21"
	}

	connection, err := rt.getConnection(hostport)
	if err != nil {
		return nil, err
	}

	var body io.ReadCloser
	body, err = connection.Retr(request.URL.Path)

	if err != nil {
		if te, ok := err.(*textproto.Error); ok {
			rt.putConnection(hostport, connection)

			if te.Code == ftp.StatusFileUnavailable {
				return &http.Response{
					Status:     "404 Not Found",
					StatusCode: 404,
					Proto:      "FTP/1.0",
					ProtoMajor: 1,
					ProtoMinor: 0,
					Request:    request,
				}, nil
			}
		}

		return nil, err
	}

	return &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Proto:      "FTP/1.0",
		ProtoMajor: 1,
		ProtoMinor: 0,
		Body: &readCloserWrapper{
			body:     body,
			rt:       rt,
			hostport: hostport,
			conn:     connection,
		},
		ContentLength: -1,
		Request:       request,
	}, nil
}
