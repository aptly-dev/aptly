package ssdb

import (
	"fmt"
	"os"
)

func ssdbLog(a ...interface{}) {
	if os.Getenv("SSDB_DEBUG") != "" {
		fmt.Println(a...)
	}
}
