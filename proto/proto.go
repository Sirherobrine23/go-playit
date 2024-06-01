package proto

import (
	"log"

	"sirherobrine23.org/playit-cloud/go-playit/logfile"
)

var debug = log.New(logfile.DebugFile, "proto.playit.gg: ", log.Ldate)