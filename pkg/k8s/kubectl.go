package k8s

import (
	"fmt"
	"os/exec"
)

const jarPathInContainer = "/deployments/app.jar"

func Copy(path, namespace, destination string) error {
	return runKubectl([]string{"cp", path, fmt.Sprintf("%s:%s", destination, jarPathInContainer), "-n", namespace}...)
}

func IsJarPresent(podName string) bool {
	return runKubectl([]string{"exec", podName, "--", "ls", jarPathInContainer}...) == nil
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
