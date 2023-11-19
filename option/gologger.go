package option

import (
	"github.com/projectdiscovery/gologger"
	"github.com/projectdiscovery/gologger/formatter"
	"github.com/projectdiscovery/gologger/levels"
)

func SetLogLevel(level levels.Level) {
	gologger.DefaultLogger.SetMaxLevel(level)
	//不带颜色打印
	gologger.DefaultLogger.SetFormatter(formatter.NewCLI(true))
	//时间戳
	gologger.DefaultLogger.SetTimestamp(true, levels.LevelFatal)
}
