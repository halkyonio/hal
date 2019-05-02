package main

import (
	"fmt"
	"github.com/snowdrop/kreate/pkg/kreate/cli/project"
)

func main() {
	createCmd := project.NewCmdProject()

	err := createCmd.Execute()
	if err != nil {
		fmt.Print(err.Error())
	}
}
