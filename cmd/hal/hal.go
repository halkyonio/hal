package main

import (
	"fmt"
	"halkyon.io/hal/pkg/hal/cli"
)

func main() {
	createCmd := cli.NewCmdHal()

	err := createCmd.Execute()
	if err != nil {
		fmt.Print(err.Error())
	}
}
