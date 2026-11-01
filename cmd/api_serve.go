package cmd

import (
	stdcontext "context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/aptly-dev/aptly/api"
	"github.com/aptly-dev/aptly/systemd/activation"
	"github.com/aptly-dev/aptly/utils"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

const (
	metricsReadHeaderTimeout = 5 * time.Second
	httpShutdownTimeout      = 30 * time.Second
)

type apiHTTPServer struct {
	name     string
	server   *http.Server
	listener net.Listener
}

type apiServeResult struct {
	name string
	err  error
}

func metricsOnlyHandler(apiHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/metrics" {
			http.NotFound(w, r)
			return
		}

		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		request := r.Clone(r.Context())
		requestURL := *r.URL
		requestURL.Path = "/api/metrics"
		requestURL.RawPath = ""
		request.URL = &requestURL
		request.RequestURI = requestURL.RequestURI()
		apiHandler.ServeHTTP(w, request)
	})
}

func validateMetricsListener(address string, metricsEnabled bool) error {
	if address != "" && !metricsEnabled {
		return errors.New("-metrics-listen requires enableMetricsEndpoint to be true")
	}
	return nil
}

func selectActivatedAPIListener(listeners []net.Listener) (net.Listener, error) {
	switch len(listeners) {
	case 0:
		return nil, nil
	case 1:
		if listeners[0] == nil {
			return nil, errors.New("systemd file descriptor is not a supported network listener")
		}
		return listeners[0], nil
	default:
		return nil, fmt.Errorf("got %d listeners from systemd; only one API listener is supported", len(listeners))
	}
}

func listenForAPI(address string) (net.Listener, error) {
	listenURL, err := url.Parse(address)
	if err == nil && listenURL.Scheme == "unix" {
		file := listenURL.Path
		_ = os.Remove(file)

		listener, err := net.Listen("unix", file)
		if err != nil {
			return nil, fmt.Errorf("failed to listen on API Unix socket %q: %w", file, err)
		}
		return listener, nil
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on API address %q: %w", address, err)
	}
	return listener, nil
}

func shutdownHTTPServers(servers []apiHTTPServer, timeout time.Duration) {
	shutdownContext, cancel := stdcontext.WithTimeout(stdcontext.Background(), timeout)
	defer cancel()

	failed := make(chan *http.Server, len(servers))
	var waitGroup sync.WaitGroup
	for _, httpServer := range servers {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			if err := httpServer.server.Shutdown(shutdownContext); err != nil {
				failed <- httpServer.server
			}
		}()
	}
	waitGroup.Wait()
	close(failed)

	for server := range failed {
		_ = server.Close()
	}
}

func serveHTTPServers(servers []apiHTTPServer, sigchan <-chan os.Signal, restoreSignals func(), waitForTasks func(), shutdownTimeout time.Duration) error {
	results := make(chan apiServeResult, len(servers))
	for _, httpServer := range servers {
		go func() {
			results <- apiServeResult{
				name: httpServer.name,
				err:  httpServer.server.Serve(httpServer.listener),
			}
		}()
	}

	serveResults := make([]apiServeResult, 0, len(servers))
	shutdownFromSignal := false
	select {
	case result := <-results:
		serveResults = append(serveResults, result)
	case <-sigchan:
		shutdownFromSignal = true
	}

	if restoreSignals != nil {
		restoreSignals()
	}

	if shutdownFromSignal {
		fmt.Printf("\nShutdown signal received, stopping HTTP servers...\n")
	} else {
		fmt.Printf("\nHTTP server stopped unexpectedly, stopping sibling server...\n")
	}
	shutdownHTTPServers(servers, shutdownTimeout)

	for len(serveResults) < len(servers) {
		serveResults = append(serveResults, <-results)
	}

	fmt.Printf("Waiting for background tasks...\n")
	waitForTasks()

	if shutdownFromSignal {
		for _, result := range serveResults {
			if result.err != nil && !errors.Is(result.err, http.ErrServerClosed) {
				return fmt.Errorf("%s server failed during shutdown: %w", result.name, result.err)
			}
		}
		return nil
	}

	result := serveResults[0]
	if result.err == nil {
		return fmt.Errorf("%s server stopped unexpectedly", result.name)
	}
	return fmt.Errorf("%s server stopped unexpectedly: %w", result.name, result.err)
}

func serveAPIWithMetrics(apiListener net.Listener, activated bool, apiAddress, metricsAddress string) error {
	var err error
	if apiListener == nil {
		apiListener, err = listenForAPI(apiAddress)
		if err != nil {
			return err
		}
	}
	defer func() { _ = apiListener.Close() }()

	metricsListener, err := net.Listen("tcp", metricsAddress)
	if err != nil {
		return fmt.Errorf("failed to listen on metrics address %q: %w", metricsAddress, err)
	}
	defer func() { _ = metricsListener.Close() }()

	apiHandler := api.Router(context)
	servers := []apiHTTPServer{
		{
			name:     "API",
			server:   &http.Server{Handler: apiHandler},
			listener: apiListener,
		},
		{
			name: "metrics",
			server: &http.Server{
				Handler:           metricsOnlyHandler(apiHandler),
				ReadHeaderTimeout: metricsReadHeaderTimeout,
			},
			listener: metricsListener,
		},
	}

	if activated {
		fmt.Printf("\nTaking over API web server at: %s (press Ctrl+C to quit)...\n", apiListener.Addr().String())
	} else {
		fmt.Printf("\nStarting API web server at: %s (press Ctrl+C to quit)...\n", apiListener.Addr().String())
	}
	fmt.Printf("Starting metrics web server at: %s...\n", metricsListener.Addr().String())

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigchan)

	restoreSignals := func() {
		signal.Reset(syscall.SIGINT, syscall.SIGTERM)
	}

	return serveHTTPServers(servers, sigchan, restoreSignals, context.TaskList().Wait, httpShutdownTimeout)
}

func aptlyAPIServe(cmd *commander.Command, args []string) error {
	var (
		err error
	)

	if len(args) != 0 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	// There are only two working options for aptly's rootDir:
	//   1. rootDir does not exist, then we'll create it
	//   2. rootDir exists and is writable
	// anything else must fail.
	// E.g.: Running the service under a different user may lead to a rootDir
	// that exists but is not usable due to access permissions.
	err = utils.DirIsAccessible(context.Config().GetRootDir())
	if err != nil {
		return err
	}

	metricsListen := context.Flags().Lookup("metrics-listen").Value.String()
	if err = validateMetricsListener(metricsListen, context.Config().EnableMetricsEndpoint); err != nil {
		return err
	}

	// Try to recycle systemd fds for listening
	listeners, err := activation.Listeners(true)
	if err == nil {
		listener, listenerErr := selectActivatedAPIListener(listeners)
		if listenerErr != nil {
			return listenerErr
		}
		if listener != nil && metricsListen != "" {
			return serveAPIWithMetrics(listener, true, "", metricsListen)
		}
		if listener != nil {
			defer func() { _ = listener.Close() }()
			fmt.Printf("\nTaking over web server at: %s (press Ctrl+C to quit)...\n", listener.Addr().String())
			err = http.Serve(listener, api.Router(context))
			if err != nil {
				return fmt.Errorf("unable to serve: %s", err)
			}
			return nil
		}
	}

	// If there are none: use the listen argument.
	listen := context.Flags().Lookup("listen").Value.String()
	if metricsListen != "" {
		return serveAPIWithMetrics(nil, false, listen, metricsListen)
	}
	fmt.Printf("\nStarting web server at: %s (press Ctrl+C to quit)...\n", listen)

	server := http.Server{Handler: api.Router(context)}

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
	go (func() {
		if _, ok := <-sigchan; ok {
			fmt.Printf("\nShutdown signal received, waiting for background tasks...\n")
			context.TaskList().Wait()
			_ = server.Shutdown(stdcontext.Background())
		}
	})()
	defer close(sigchan)

	listenURL, err := url.Parse(listen)
	if err == nil && listenURL.Scheme == "unix" {
		file := listenURL.Path
		_ = os.Remove(file)

		var listener net.Listener
		listener, err = net.Listen("unix", file)
		if err != nil {
			return fmt.Errorf("failed to listen on: %s\n%s", file, err)
		}
		defer func() { _ = listener.Close() }()

		err = server.Serve(listener)
	} else {
		server.Addr = listen
		err = server.ListenAndServe()
	}

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("unable to serve: %s", err)
	}

	return nil
}

func makeCmdAPIServe() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyAPIServe,
		UsageLine: "serve",
		Short:     "start API HTTP service",
		Long: `
Start HTTP server with aptly REST API. The server can listen to either a port
or Unix domain socket. When using a socket, Aptly will fully manage the socket
file. This command also supports taking over from a systemd file descriptors to
enable systemd socket activation.

An optional metrics-only TCP listener can be enabled with -metrics-listen. It
serves GET and HEAD requests at /metrics and requires enableMetricsEndpoint in
the aptly configuration. This listener uses plain HTTP; provide access control
and TLS through network policy or a proxy when required.

Example:

  $ aptly api serve -listen=:8080
  $ aptly api serve -listen=unix:///tmp/aptly.sock
  $ aptly api serve -listen=:8080 -metrics-listen=127.0.0.1:9090
`,
		Flag: *flag.NewFlagSet("aptly-serve", flag.ExitOnError),
	}

	cmd.Flag.String("listen", ":8080", "host:port for HTTP listening or unix://path to listen on a Unix domain socket")
	cmd.Flag.String("metrics-listen", "", "host:port for optional metrics-only HTTP listening (requires enableMetricsEndpoint)")
	cmd.Flag.Bool("no-lock", false, "don't lock the database")

	return cmd

}
