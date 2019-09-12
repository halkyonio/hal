package cmdutil

func CommandName(name, fullParentName string) string {
	return fullParentName + " " + name
}
