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

	fmt.Println(to_string(reflect.ValueOf(config).Elem()))
	return err

}

func to_string(v reflect.Value) string {

	switch v.Kind() {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.Slice:
		var str_slice []string
		for i := 0; i < v.Len(); i++ {
			str_slice = append(str_slice, to_string(v.Index(i)))
		}
		return strings.Join(str_slice, ", ")
	case reflect.Struct:
		var str_slice []string
		typ := reflect.TypeOf(v.Interface())
		for i := 0; i < typ.NumField(); i++ {
			str_slice = append(str_slice, typ.Field(i).Name+": "+to_string(v.Field(i)))
		}
		return strings.Join(str_slice, "\n")
	//case reflect.Map:
	//	var str_slice []string

	case reflect.String:
		return v.String()
	}
	return ""

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
