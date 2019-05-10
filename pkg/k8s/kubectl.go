package k8s

import (
	"fmt"
	"os/exec"
)

func Copy(path, namespace, destination string) error {
	return run("cp", []string{path, fmt.Sprintf("%s:/deployments/app.jar", destination), "-n", namespace}...)
}

func Apply(path, namespace string) error {
	return run("kubectl", []string{"apply", "-f", path, "-n", namespace}...)
}

func run(cmdName string, args ...string) error {
	command := exec.Command(cmdName, args...)
	err := command.Run()
	if err != nil {
		return err
	}
	return nil
}
