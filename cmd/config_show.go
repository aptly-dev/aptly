package cmd

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)
import "github.com/smira/commander"

func aptlyConfigShow(cmd *commander.Command, args []string) error {

	config := context.Config()

	fmt.Println(to_string(reflect.ValueOf(config).Elem(), 0))

	return nil

}

func to_string(v reflect.Value, tabs int) string {

	switch v.Kind() {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.Slice:
		var str_slice []string
		for i := 0; i < v.Len(); i++ {
			str_slice = append(str_slice, to_string(v.Index(i), tabs))
		}
		return strings.Join(str_slice, ", ")
	case reflect.Struct:
		var str_slice []string
		typ := reflect.TypeOf(v.Interface())
		//str_slice = append(str_slice, make_tabs(tabs)+"{")
		for i := 0; i < typ.NumField(); i++ {
			str_slice = append(str_slice, make_tabs(tabs)+typ.Field(i).Name+": "+to_string(v.Field(i), tabs+1))
		}
		//str_slice = append(str_slice, make_tabs(tabs)+"}")
		return strings.Join(str_slice, "\n")
	case reflect.Map:
		var str_slice []string
		str_slice = append(str_slice, "")
		for _, key := range v.MapKeys() {
			str_slice = append(str_slice, make_tabs(tabs)+"- "+to_string(key, tabs)+":\n"+to_string(v.MapIndex(key), tabs+1))
		}
		//str_slice = append(str_slice, make_tabs(tabs)+"}")
		return strings.Join(str_slice, "\n")
	case reflect.String:
		return v.String()
	}
	return ""

}

func make_tabs(tabs int) string {
	str := ""
	for i := 0; i < tabs; i++ {
		str += "\t"
	}
	return str
}

func makeCmdConfigShow() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyConfigShow,
		UsageLine: "show",
		Short:     "show current aptly's config",
		Long: `
Command show displays the current aptly configuration.

Example:

  $ aptly config show

`,
	}
	return cmd
}
