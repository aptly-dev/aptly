package cmd

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/smira/commander"
)

func aptlyConfigShow(cmd *commander.Command, args []string) error {

	config := context.Config()

	config_to_string := toString(reflect.ValueOf(config).Elem())

	if config_to_string == "" {
		return fmt.Errorf("Error processing configuration")
	}

	fmt.Println(config_to_string)

	return nil
}

func toString(v reflect.Value) string {

	switch v.Kind() {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.Slice:
		var str_slice []string
		for i := 0; i < v.Len(); i++ {
			str_slice = append(str_slice, toString(v.Index(i)))
		}
		return strings.Join(str_slice, ", ")
	case reflect.Struct:
		var str_slice []string
		typ := reflect.TypeOf(v.Interface())
		for i := 0; i < typ.NumField(); i++ {
			str_slice = append(str_slice, typ.Field(i).Name+": "+toString(v.Field(i)))
		}
		return strings.Join(str_slice, "\n")
	case reflect.Map:
		var str_slice []string
		str_slice = append(str_slice, "")
		for _, key := range v.MapKeys() {
			str_slice = append(str_slice, "- "+toString(key)+":\n"+toString(v.MapIndex(key)))
		}
		return strings.Join(str_slice, "\n")
	case reflect.String:
		return v.String()
	default:
		return ""
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
