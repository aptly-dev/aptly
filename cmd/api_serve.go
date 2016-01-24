package cmd

import (
	"fmt"
	"github.com/smira/aptly/api"
	"github.com/smira/commander"
	"github.com/smira/flag"
	"net/http"
)

func aptlyAPIServe(cmd *commander.Command, args []string) error {
	var (
		err error
	)

	if len(args) != 0 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	listen := context.Flags().Lookup("listen").Value.String()

	fmt.Printf("\nStarting web server at: %s (press Ctrl+C to quit)...\n", listen)

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
Stat HTTP server with aptly REST API.

Example:

  $ aptly api serve -listen=:8080
`,
		Flag: *flag.NewFlagSet("aptly-serve", flag.ExitOnError),
	}

	cmd.Flag.String("listen", ":8080", "host:port for HTTP listening")
	cmd.Flag.Bool("no-lock", false, "don't lock the database")

	return cmd

}
