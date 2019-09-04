package main

import (
	"fmt"
	"halkyon.io/hal/pkg/hal/cli"
)

func main() {
	createCmd := cli.NewCmdKreate()

	err := createCmd.Execute()
	if err != nil {
		fmt.Print(err.Error())
	}
}
