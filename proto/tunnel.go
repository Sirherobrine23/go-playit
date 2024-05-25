package proto

import (
	"log"
	"os"
)

// Write log and show in terminal to debug
var logDebug *log.Logger = log.New(os.Stderr, "plait.gg", log.Ltime|log.Ldate)
