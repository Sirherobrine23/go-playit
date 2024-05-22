package tunnel

import (
	"log"
	"os"
)

var LogDebug = log.New(os.Stderr, "go-playit.gg: ", log.Ldate|log.Ltime)