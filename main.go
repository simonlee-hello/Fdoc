package main

import (
	"Fdoc/option"
	"Fdoc/pkg"
	"github.com/projectdiscovery/gologger"
	"github.com/projectdiscovery/gologger/levels"
	"time"
)

func main() {

	info := &option.FlagInfo{}
	info.InitFlag()
	option.SetLogLevel(levels.LevelWarning)

	startTime := time.Now()
	pkg.WalkAndCompress(info)
	duration := time.Since(startTime)
	gologger.Info().Msgf("Execution time: %v\n", duration)
}
