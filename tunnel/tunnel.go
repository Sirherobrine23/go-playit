package tunnel

import (
	"log"

	"sirherobrine23.org/playit-cloud/go-playit/logfile"
)

var debug = log.New(logfile.DebugFile, "tunnel.playit.gg: ", log.Ldate)