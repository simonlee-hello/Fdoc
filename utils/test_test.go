package utils

import (
	"fmt"
	"testing"
)

func TestName(t *testing.T) {
	res := IsSymlink("/Users/simon/Downloads/IDAPro/dbgsrv")
	fmt.Println(res)
}
