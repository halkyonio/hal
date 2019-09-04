package main

import (
	"fmt"
	"halkyon.io/kreate/pkg/kreate/cli"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	createCmd := cli.NewCmdKreate(version, commit, date)

	err := createCmd.Execute()
	if err != nil {
		fmt.Print(err.Error())
	}
}
