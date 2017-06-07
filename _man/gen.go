package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/smira/aptly/cmd"
	"github.com/smira/commander"
	"github.com/smira/flag"
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

var authorsS string

func authors() string {
	return authorsS
}

func main() {
	command := cmd.RootCommand()
	command.UsageLine = "aptly"
	command.Dispatch(nil)

	_File, _ := filepath.Abs("./man")

	templ := template.New("man").Funcs(template.FuncMap{
		"allFlags":    allFlags,
		"findCommand": findCommand,
		"toUpper":     strings.ToUpper,
		"capitalize":  capitalize,
		"authors":     authors,
	})
	template.Must(templ.ParseFiles(filepath.Join(filepath.Dir(_File), "aptly.1.ronn.tmpl")))

	authorsF, err := os.Open(filepath.Join(filepath.Dir(_File), "..", "AUTHORS"))
	if err != nil {
		log.Fatal(err)
	}

	authorsB, err := ioutil.ReadAll(authorsF)
	if err != nil {
		log.Fatal(err)
	}

	authorsF.Close()

	authorsS = string(authorsB)

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
