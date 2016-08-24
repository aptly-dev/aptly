package cmd

import (
	"fmt"
	"io/ioutil"
	"strings"
	"encoding/json"
	"github.com/smira/commander"
)

type AptlyFilterStruct struct {
	Name				string		`json:"name"`
	Version			string		`json:"version"`
}

type aptlySetupConfigStruct struct {
	Mirrors					[]AptlyMirrorStruct
}

type AptlyMirrorStruct struct {
	Name						string							`json:"name"`
	Url							string							`json:"url"`
	Dist 						string							`json:"dist"`
	Component				string 							`json:"component"`
	Filter					[]AptlyFilterStruct `json:"filter"`
	FilterDeps			bool								`json:"filter-with-deps"`
}

func createStringArray(array ...string) []string{

	var strings_arr []string

	for _, e := range array {
		if e != "" {
			strings_arr = append(strings_arr, e)
		}
	}
	return strings_arr
}

func isEmpty(element string) bool {
	if element == "" {
		return true
	}
	return false
}

func mirrorExists(mirror_name string) bool {
	 mirror, _ := context.CollectionFactory().RemoteRepoCollection().ByName(mirror_name)

	 if mirror == nil {
		 return false
	 }
	 return true
}

func repoExists(repo_name string) bool {
	 repo, _ := context.CollectionFactory().LocalRepoCollection().ByName(repo_name)

	 if repo == nil {
		 return false
	 }
	 return true
}

func (filter *AptlyFilterStruct) createAptlyMirrorFilter() string {

	var f []string
	if !isEmpty(filter.Name) {
		f = append(f, fmt.Sprintf("Name (= %s)", filter.Name))
	}

  if !isEmpty(filter.Version) {
		f = append(f, fmt.Sprintf("$Version (= %s)", filter.Version))
	}

	f_str := ""

	if len(f) > 1 {
		f_str = fmt.Sprintf("( %s )", strings.Join(f, " , "))
	} else if len(f) == 1 {
		f_str = fmt.Sprintf("( %s )", f[0])
	}

	return f_str

}

func (mirror *AptlyMirrorStruct) createAptlyMirror() ([]string, error) {

	filter_with_deps_cmd := ""
	filter_cmd := ""

	if isEmpty(mirror.Name) {
		return nil, fmt.Errorf("Missing name from mirror")
	}

	if isEmpty(mirror.Url) {
		return nil, fmt.Errorf("Missing url from mirror")
	}

	if isEmpty(mirror.Dist) {
		return nil, fmt.Errorf("Missing distribution from mirror")
	}

	component := mirror.Component
	if isEmpty(component) {
		component = ""
	}

	if mirror.Filter != nil {

		var filter_cmds []string
		for _, filter := range mirror.Filter {
			filter_cmds = append(filter_cmds, filter.createAptlyMirrorFilter())
		}

		if len(filter_cmds) > 1 {
			filter_cmd = fmt.Sprintf("-filter=%s", strings.Join(filter_cmds, " | "))
		} else if len(filter_cmds) == 1 {
			filter_cmd = fmt.Sprintf("-filter=%s", filter_cmds[0])
		}
	}

	if mirror.FilterDeps {
		filter_with_deps_cmd = "-filter-with-deps"
	}

	args := createStringArray("mirror", "create", filter_cmd, filter_with_deps_cmd, mirror.Name, mirror.Url, mirror.Dist, component)

	fmt.Println(args)

	return args, nil
}

func (mirror *AptlyMirrorStruct) updateAptlyMirror() ([]string, error) {

	if isEmpty(mirror.Name) {
		return nil, fmt.Errorf("Missing name from mirror")
	}

	args := createStringArray("mirror", "update", mirror.Name)
	return args, nil

}


func (m *aptlySetupConfigStruct) createAndUpdateMirrors() ([][]string, error) {

	var commands [][]string
	var cmd_create []string
	var cmd_update []string
	var e error

	for _, mirror := range m.Mirrors {

		if ! mirrorExists(mirror.Name) {
			cmd_create, e = mirror.createAptlyMirror()
			if e != nil {
				return nil, e
			}
			commands = append(commands, cmd_create)
		}

		cmd_update, e = mirror.updateAptlyMirror()
		if e != nil {
			return nil, e
		}
		commands = append(commands, cmd_update)
	}

	return commands, nil

}

func aptlyRunSetup(cmd *commander.Command, args []string) error {

	// Get setup configuration
	filename := context.Config().SetupFile

	f, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("Unable to read file: %s", err)
	}

	var mirrors aptlySetupConfigStruct

	json.Unmarshal(f, &mirrors)

	commands, err := mirrors.createAndUpdateMirrors()
	if err != nil {
		return err
	}

 	err = aptlyTaskRunCommands(commands)

	/*if returnCode != 0 {
		}
		return fmt.Errorf("at least one command has reported an error")
	}*/

  return err
}


func makeCmdRunSetup() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyRunSetup,
		UsageLine: "setup",
		Short:     "setup mirrors and repos from a configuration file",
		Long: `
Initialise or update mirrors and repos defined in a configuration file referenced
in aptly.conf.

ex:
  $ aptly run setup
`,
  }
	return cmd
}
