package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-shellwords"
	"github.com/smira/commander"
)

func aptlyTaskRun(cmd *commander.Command, args []string) error {

	var err error
	var cmd_list [][]string

	if filename := cmd.Flag.Lookup("filename").Value.Get().(string); filename != "" {
		var text string
		cmd_args := []string{}

		if finfo, err := os.Stat(filename); os.IsNotExist(err) || finfo.IsDir() {
			return fmt.Errorf("No such file, %s\n", filename)
		}

		fmt.Println("Reading file...\n")

		file, err := os.Open(filename)

		if err != nil {
			return err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)

		for scanner.Scan() {
			text = strings.TrimSpace(scanner.Text()) + ","
			parsed_args, _ := shellwords.Parse(text)
			cmd_args = append(cmd_args, parsed_args...)
		}

		if err = scanner.Err(); err != nil {
			return err
		}

		if len(cmd_args) == 0 {
			return fmt.Errorf("The file is empty. Exiting...\n")
		}

		cmd_list = formatCommands(cmd_args)

	} else if len(args) == 0 {
		var text string
		cmd_args := []string{}

		fmt.Println("Please enter one command per line and leave one blank when finished.")

		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Printf("> ")
			text, _ = reader.ReadString('\n')
			if text == "\n" {
				break
			} else {
				text = strings.TrimSpace(text) + ","
				parsed_args, _ := shellwords.Parse(text)
				cmd_args = append(cmd_args, parsed_args...)
			}
		}

		if len(cmd_args) == 0 {
			return fmt.Errorf("Nothing entered. Exiting...\n")
		}

		cmd_list = formatCommands(cmd_args)

	} else {
		cmd_list = formatCommands(args)
	}

	commandErrored := false

	for i, command := range cmd_list {
		if !commandErrored {
			context.Progress().ColoredPrintf("@g%d) [Running]: %s@!", (i + 1), strings.Join(command, " "))
			context.Progress().ColoredPrintf("\n@yBegin command output: ----------------------------@!")
			context.Progress().Flush()

			returnCode := Run(RootCommand(), command, false)
			if returnCode != 0 {
				commandErrored = true
			}
			context.Progress().ColoredPrintf("\n@yEnd command output: ------------------------------@!")
			CleanupContext()
		} else {
			context.Progress().ColoredPrintf("@r%d) [Skipping]: %s@!", (i + 1), strings.Join(command, " "))
		}
	}

	if commandErrored {
		err = fmt.Errorf("At least one command has reported an error\n")
	}

	return err
}

func formatCommands(args []string) [][]string {

	var cmd []string
	var cmd_array [][]string

	for _, s := range args {
		if s_trimmed := strings.TrimRight(s, ","); s_trimmed != s {
			cmd = append(cmd, s_trimmed)
			cmd_array = append(cmd_array, cmd)
			cmd = []string{}
		} else {
			cmd = append(cmd, s)
		}
	}

	if len(cmd) > 0 {
		cmd_array = append(cmd_array, cmd)
	}

	return cmd_array
}

func makeCmdTaskRun() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyTaskRun,
		UsageLine: "run -filename=<filename> | <command1>, <command2>, ...",
		Short:     "run aptly tasks",
		Long: `
Command helps origanise multiple aptly commands in one single aptly task, running as single thread.

Example:

  $ aptly task run
  > repo create local
  > repo add local pkg1
  > publish repo local
  > serve
  >

`,
	}

	cmd.Flag.String("filename", "", "specifies the filename that contains the commands to run")
	return cmd
}
