package main

import (
	"fmt"
	"halkyon.io/kreate/pkg/kreate/cli"
)

func main() {
	createCmd := cli.NewCmdKreate()

	err := createCmd.Execute()
	if err != nil {
		fmt.Print(err.Error())
	}
}
