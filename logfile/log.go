package logfile

import (
	"encoding/json"
	"os"
)

var DebugFile = func() *os.File {
	file, err := os.Create("./debug.log")
	if err != nil {
		panic(err)
	}
	return file
}()

func JSONString(data any) string {
	d, _ := json.Marshal(data)
	return string(d)
}