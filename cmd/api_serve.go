package cmd

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"

	"github.com/smira/aptly/api"
	"github.com/smira/aptly/systemd/activation"
	"github.com/smira/aptly/utils"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

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
	err = utils.DirIsAccessible(context.Config().RootDir)
	if err != nil {
		return err
	}

	// Try to recycle systemd fds for listening
	listeners, err := activation.Listeners(true)
	if len(listeners) > 1 {
		panic("Got more than 1 listener from systemd. This is currently not supported!")
	}
	if err == nil && len(listeners) == 1 {
		listener := listeners[0]
		defer listener.Close()
		fmt.Printf("\nTaking over web server at: %s (press Ctrl+C to quit)...\n", listener.Addr().String())
		err = http.Serve(listener, api.Router(context))
		if err != nil {
			return fmt.Errorf("unable to serve: %s", err)
		}
		return nil
	}

	// If there are none: use the listen argument.
	listen := context.Flags().Lookup("listen").Value.String()
	fmt.Printf("\nStarting web server at: %s (press Ctrl+C to quit)...\n", listen)

	listenURL, err := url.Parse(listen)
	if err == nil && listenURL.Scheme == "unix" {
		file := listenURL.Path
		os.Remove(file)

		var listener net.Listener
		listener, err = net.Listen("unix", file)
		if err != nil {
			return fmt.Errorf("failed to listen on: %s\n%s", file, err)
		}
		defer listener.Close()

		err = http.Serve(listener, api.Router(context))
		if err != nil {
			return fmt.Errorf("unable to serve: %s", err)
		}
		return nil
	}

	err = http.ListenAndServe(listen, api.Router(context))
	if err != nil {
		return fmt.Errorf("unable to serve: %s", err)
	}

	return err
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

Example:

  $ aptly api serve -listen=:8080
  $ aptly api serve -listen=unix:///tmp/aptly.sock
`,
		Flag: *flag.NewFlagSet("aptly-serve", flag.ExitOnError),
	}

	cmd.Flag.String("listen", ":8080", "host:port for HTTP listening or unix://path to listen on a Unix domain socket")
	cmd.Flag.Bool("no-lock", false, "don't lock the database")

	return cmd

}
