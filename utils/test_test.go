package utils

import (
	"testing"
)

func TestName(t *testing.T) {
	//res := IsSymlink("/Users/simon/Downloads/IDAPro/dbgsrv")
	//fmt.Println(res)

	//password := "your_password_here" // 替换为你的密码
	FilesToTarGz("", "/Users/simon/Desktop/out.tar.gz", []string{"/Users/simon/Desktop/tmp/1.txt", "/Users/simon/Desktop/china_user.txt"})

}
