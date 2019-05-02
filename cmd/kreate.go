package main

import (
	"fmt"
	"github.com/snowdrop/kreate/pkg/kreate/cli"
)

func main() {
	createCmd := cli.NewCmdKreate()

	err := createCmd.Execute()
	if err != nil {
		fmt.Print(err.Error())
	}
}
