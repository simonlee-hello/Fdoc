package main

import (
	"Fdoc/option"
	"Fdoc/pkg"
	"github.com/projectdiscovery/gologger/levels"
)

func main() {

	info := &option.FlagInfo{}
	info.InitFlag()
	option.SetLogLevel(levels.LevelWarning)

	pkg.WalkAndCompress(info)

}
