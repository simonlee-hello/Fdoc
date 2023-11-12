package main

import (
	"Fdoc/pkg"
	"github.com/projectdiscovery/gologger"
	"github.com/projectdiscovery/gologger/levels"
)

func main() {

	//gologger.DefaultLogger.SetMaxLevel(levels.LevelDebug)
	gologger.DefaultLogger.SetMaxLevel(levels.LevelInfo)

	pkg.QueryAndZip()
}
