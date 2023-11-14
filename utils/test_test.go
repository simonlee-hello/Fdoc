package utils

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/projectdiscovery/gologger"
	"github.com/projectdiscovery/gologger/levels"
)

func TestName(t *testing.T) {
	//res := IsLink("/Users/simon/Downloads/IDAPro/dbgsrv")
	//fmt.Println(res)

	//password := "your_password_here" // 替换为你的密码
	//FilesToTarGz("", "/Users/simon/Desktop/out.tar.gz", []string{"/Users/simon/Desktop/tmp/1.txt", "/Users/simon/Desktop/china_user.txt"})
	path := "/Users/simon/Downloads/IDAPro/dbgsrv"
	file, _ := os.Open(path)
	fmt.Println(file.Stat())
	//fmt.Println(IsSymlink(path))
}

func TestLogger(t *testing.T) {
	gologger.DefaultLogger.SetMaxLevel(levels.LevelDebug)
	//	gologger.DefaultLogger.SetFormatter(&formatter.JSON{})
	gologger.Info().Msgf("\tgologger: sample test\t\n")
	gologger.Info().Str("user", "pdteam").Msg("running simulation program")
	for i := 0; i < 10; i++ {
		gologger.Info().Str("count", strconv.Itoa(i)).Msg("running simulation step...")
	}
	gologger.Debug().Str("state", "running").Msg("planner running")
	gologger.Warning().Str("state", "errored").Str("status", "404").Msg("could not run")
	gologger.Fatal().Msg("bye bye")
}

func TestStat(t *testing.T) {
	fileinfo, err := os.Stat("/Users/simon/Downloads/IDAPro/dbgsrv")
	fileinfo2, err := os.Lstat("/Users/simon/Downloads/IDAPro/dbgsrv")
	fmt.Println(fileinfo)
	fmt.Println("NAME:", fileinfo.Name())
	fmt.Println("MODE:", fileinfo.Mode())
	fmt.Println("SIZE:", fileinfo.Size())
	fmt.Println("ISDIR:", fileinfo.IsDir())
	fmt.Println("MODTIME:", fileinfo.ModTime())
	fmt.Println("SYS:", fileinfo.Sys())

	fmt.Println("--------")
	fmt.Println(fileinfo2)
	fmt.Println("NAME:", fileinfo2.Name())
	fmt.Println("MODE:", fileinfo2.Mode())
	fmt.Println("SIZE:", fileinfo2.Size())
	fmt.Println("ISDIR:", fileinfo2.IsDir())
	fmt.Println("MODTIME:", fileinfo2.ModTime())
	fmt.Println("SYS:", fileinfo2.Sys())
	if err != nil {
		fmt.Println(err)
	}
}
