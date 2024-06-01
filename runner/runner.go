package runner

import (
	"log"

	"sirherobrine23.org/playit-cloud/go-playit/logfile"
)

var debug = log.New(logfile.DebugFile, "runner.playit.gg: ", log.Ldate)