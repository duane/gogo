package main

import "os/exec"
import "strings"

func Uname() (os string, release string, arch string) {
	uname := exec.Command("uname", "-srm")
	outBytes, err := uname.Output()
	if err != nil {
		return "unknown", "unknown", "unknown"
	}
	toks := strings.Split(strings.Trim(string(outBytes), "\t\r\n "), " ")

	return toks[0], toks[1], toks[2]
}
