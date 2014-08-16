package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-shellwords"
	"github.com/smira/commander"
	"github.com/wsxiaoys/terminal/color"
)

func aptlyTaskRun(cmd *commander.Command, args []string) error {

	var err error
	var cmd_list [][]string

	if len(args) == 0 {
		var text string
		cmd_args := []string{}

		fmt.Println("One command per line and press enter when finished.")

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

	} else if len(args) == 1 {
		var text string
		cmd_args := []string{}

		if finfo, err := os.Stat(args[0]); os.IsNotExist(err) || finfo.IsDir() {
			return fmt.Errorf("No such file, %s\n", args[0])
		}

		fmt.Println("Reading file...\n")

		file, err := os.Open(args[0])

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

	} else if len(args) > 1 {
		cmd_list = formatCommands(args)
	}

	switchContext()

	for i, command := range cmd_list {

		if !context.panicked {
			color.Printf("@g%d) [Running]: %s@!\n", (i + 1), strings.Join(command, " "))
			color.Println("\n@yBegin command output: ----------------------------\n@!")
			Run(command, false)
			color.Println("\n@yEnd command output: ------------------------------\n@!")

		} else {
			color.Printf("@r%d) [Skipping]: %s@!\n", (i + 1), strings.Join(command, " "))
		}

	}

	if context.panicked {
		err = fmt.Errorf("At least one command has reported an error\n")
	}

	switchContext()

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
		UsageLine: "run <filename> | <command1>, <command2>, ...",
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

	return cmd
}
