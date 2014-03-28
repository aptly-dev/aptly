package main

import (
	"fmt"
	"github.com/smira/aptly/cmd"
	"github.com/smira/commander"
	"github.com/smira/flag"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
)

func allFlags(flags *flag.FlagSet) []*flag.Flag {
	result := []*flag.Flag{}
	flags.VisitAll(func(f *flag.Flag) {
		result = append(result, f)
	})
	return result
}

func findCommand(cmd *commander.Command, name string) (*commander.Command, error) {
	for _, c := range cmd.Subcommands {
		if c.Name() == name {
			return c, nil
		}
	}

	return nil, fmt.Errorf("command %s not found", name)
}

func capitalize(s string) string {
	parts := strings.Split(s, " ")
	for i, part := range parts {
		if part[0] != '<' && part[0] != '[' && part[len(part)-1] != '>' && part[len(part)-1] != ']' {
			parts[i] = "`" + part + "`"
		}
	}

	return strings.Join(parts, " ")
}

func main() {
	command := cmd.RootCommand()
	command.UsageLine = "aptly"
	command.Dispatch(nil)

	_, _File, _, _ := runtime.Caller(0)
	_File, _ = filepath.Abs(_File)

	templ := template.New("man").Funcs(template.FuncMap{
		"allFlags":    allFlags,
		"findCommand": findCommand,
		"toUpper":     strings.ToUpper,
		"capitalize":  capitalize,
	})
	template.Must(templ.ParseFiles(filepath.Join(filepath.Dir(_File), "aptly.1.ronn.tmpl")))

	output, err := os.Create(filepath.Join(filepath.Dir(_File), "aptly.1.ronn"))
	if err != nil {
		log.Fatal(err)
	}

	err = templ.ExecuteTemplate(output, "main", command)
	if err != nil {
		log.Fatal(err)
	}

	output.Close()

	out, err := exec.Command("ronn", filepath.Join(filepath.Dir(_File), "aptly.1.ronn")).CombinedOutput()
	if err != nil {
		os.Stdout.Write(out)
		log.Fatal(err)
	}

	cmd := exec.Command("man", filepath.Join(filepath.Dir(_File), "aptly.1"))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
