package network

import (
	"log"

	"sirherobrine23.org/playit-cloud/go-playit/logfile"
)

var debug = log.New(logfile.DebugFile, "network.playit.gg: ", log.Ldate)