package main

import (
	"Fdoc/cmd"
	"Fdoc/pkg"
	"fmt"
)

func main() {

	info := pkg.GetFlag()
	fmt.Println(info)

	cmd.QueryAndZip(info)
}
