package main

import (
	"Fdoc/option"
	"Fdoc/pkg"
	"github.com/projectdiscovery/gologger"
	"github.com/projectdiscovery/gologger/formatter"
	"github.com/projectdiscovery/gologger/levels"
)

func main() {

	//gologger.DefaultLogger.SetMaxLevel(levels.LevelDebug)
	gologger.DefaultLogger.SetMaxLevel(levels.LevelWarning)
	//不带颜色打印
	gologger.DefaultLogger.SetFormatter(formatter.NewCLI(true))
	//gologger.DefaultLogger.SetTimestamp(true, levels.LevelDebug)
	gologger.DefaultLogger.SetTimestamp(true, levels.LevelFatal)

	info := &option.FlagInfo{}
	info.InitFlag()
	gologger.Info().Msg("Starting..")
	pkg.WalkAndCompress(info)
	gologger.Info().Msg("Exiting..")
}
