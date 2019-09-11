package main

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"halkyon.io/hal/pkg/hal/cli"
	"os"
)

func main() {
	var clidoc = &cobra.Command{
		Use:   "hal-doc",
		Short: "Generate CLI reference for hal",

		Example: `  # Generate a markdown-formatted CLI reference page for hal 
  hal-doc reference > cli-reference.adoc`,
		Args:      cobra.OnlyValidArgs,
		ValidArgs: []string{"help", "reference", "structure"},

		Run: func(command *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Print(command.Usage())
			} else {
				switch args[0] {
				case "reference":
					fmt.Println(referencePrinter(cli.NewCmdHal(), 0))
				case "structure":
                	fmt.Print(commandPrinter(cli.NewCmdHal(), 0))
				default:
					fmt.Print(command.Usage())
				}
			}
		},
	}
	err2 := clidoc.Execute()
	if err2 != nil {
		fmt.Println(errors.Cause(err2))
		os.Exit(1)
	}

}

// Uses portions of the help / cmd outputter in cobra13 as part of a CLI reference guide and outputs each command
func referenceCommandFormatter(command *cobra.Command) string {

	// Get length

	var spacer string
	for i := 0; i < len(command.Name()); i++ {
		spacer = spacer + "~"
	}

	return fmt.Sprintf(`[[%s]]
%s
%s

[source,sh]
----
%s
----

_________________
Example using %s
_________________

[source,sh]
----
%s
----

%s

`,
		command.Name(),
		command.Name(),
		spacer,
		command.Use,
		command.Name(),
		command.Example,
		command.Long)
}

// This prints out the CLI reference
func referencePrinter(command *cobra.Command, level int) string {

	// Table generation
	commandListTable := `
.List of Commands
[width="100%",cols="21%,79%",options="header",]
|===
| Name | Description

`
	for _, subcommand := range command.Commands() {
		commandListTable = commandListTable + fmt.Sprintf("| link:#%s[%s]\n| %s\n\n", subcommand.Name(), subcommand.Name(), subcommand.Short)
	}

	commandListTable = commandListTable + "|==="

	// Create documentation for each command
	var commandOutput string
	for _, subcommand := range command.Commands() {
		commandOutput = commandOutput + referenceCommandFormatter(subcommand)
	}

	// The main markdown "template" for everything
	return fmt.Sprintf(`= Overview of the hal CLI Structure

___________________
Example application
___________________

[source,sh]
----
%s 
----

%s

[[syntax]]
Syntax
------

%s

[[cli-structure]]
CLI Structure
+++++++++++++

[source,sh]
----
%s
----

%s
`,
		command.Example,
		command.Long,
		commandListTable,
		fmt.Sprint(commandPrinter(command, 0)),
		commandOutput)
}

func commandPrinter(command *cobra.Command, level int) string {
	var finalCommand string
	// add indentation
	for i := 0; i < level; i++ {
		finalCommand = finalCommand + "    "
	}
	finalCommand = finalCommand +
		command.Name() +
		" " +
		flattenFlags(getFlags(command.NonInheritedFlags())) +
		": " +
		command.Short +
		"\n"
	for _, subcommand := range command.Commands() {
		finalCommand = finalCommand + commandPrinter(subcommand, level+1)
	}
	return finalCommand
}

func flattenFlags(flags []string) string {
	var flagString string
	for _, flag := range flags {
		flagString = flagString + flag + " "
	}
	return flagString
}

func getFlags(flags *pflag.FlagSet) []string {
	var f []string
	flags.VisitAll(func(flag *pflag.Flag) {
		f = append(f, fmt.Sprintf("--%v", flag.Name))
	})
	return f
}
