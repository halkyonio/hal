package k8s

import (
	"fmt"
	"os/exec"
)

func Copy(path, namespace, destination string) error {
	return runKubectl([]string{"cp", path, fmt.Sprintf("%s:/deployments/app.jar", destination), "-n", namespace}...)
}

func Apply(path, namespace string) error {
	return runKubectl([]string{"apply", "-f", path, "-n", namespace}...)
}

func runKubectl(args ...string) error {
	command := exec.Command("kubectl", args...)
	err := command.Run()
	if err != nil {
		return err
	}
	return nil
}
