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
	var cmdList [][]string

	if filename := cmd.Flag.Lookup("filename").Value.Get().(string); filename != "" {
		var text string
		cmdArgs := []string{}

		var finfo os.FileInfo
		if finfo, err = os.Stat(filename); os.IsNotExist(err) || finfo.IsDir() {
			return fmt.Errorf("no such file, %s", filename)
		}

		fmt.Print("Reading file...\n\n")

		var file *os.File
		file, err = os.Open(filename)

		if err != nil {
			return err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)

		for scanner.Scan() {
			text = strings.TrimSpace(scanner.Text()) + ","
			parsedArgs, _ := shellwords.Parse(text)
			cmdArgs = append(cmdArgs, parsedArgs...)
		}

		if err = scanner.Err(); err != nil {
			return err
		}

		if len(cmdArgs) == 0 {
			return fmt.Errorf("the file is empty")
		}

		cmdList = formatCommands(cmdArgs)
	} else if len(args) == 0 {
		var text string
		cmdArgs := []string{}

		fmt.Println("Please enter one command per line and leave one blank when finished.")

		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Printf("> ")
			text, _ = reader.ReadString('\n')
			if text == "\n" {
				break
			}
			text = strings.TrimSpace(text) + ","
			parsedArgs, _ := shellwords.Parse(text)
			cmdArgs = append(cmdArgs, parsedArgs...)
		}

		if len(cmdArgs) == 0 {
			return fmt.Errorf("nothing entered")
		}

		cmdList = formatCommands(cmdArgs)
	} else {
		cmdList = formatCommands(args)
	}

	commandErrored := false

	for i, command := range cmdList {
		if !commandErrored {
			err = context.ReOpenDatabase()
			if err != nil {
				return fmt.Errorf("failed to reopen DB: %s", err)
			}
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
		err = fmt.Errorf("at least one command has reported an error")
	}

	return err
}

func formatCommands(args []string) [][]string {
	var cmd []string
	var cmdArray [][]string

	for _, s := range args {
		if sTrimmed := strings.TrimRight(s, ","); sTrimmed != s {
			cmd = append(cmd, sTrimmed)
			cmdArray = append(cmdArray, cmd)
			cmd = []string{}
		} else {
			cmd = append(cmd, s)
		}
	}

	if len(cmd) > 0 {
		cmdArray = append(cmdArray, cmd)
	}

	return cmdArray
}

func makeCmdTaskRun() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyTaskRun,
		UsageLine: "run -filename=<filename> | <command1>, <command2>, ...",
		Short:     "run aptly tasks",
		Long: `
Command helps organise multiple aptly commands in one single aptly task, running as single thread.

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
