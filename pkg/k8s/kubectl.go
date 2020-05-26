package k8s

import (
	"fmt"
	"halkyon.io/hal/pkg/log"
	"os"
	"os/exec"
)

const (
	JarPathInContainer             = "/deployments/"
	SourcePathInContainer          = "/usr/src/component.tar"
	ExtractedSourcePathInContainer = "/usr/src"
)

var kubectl = "kubectl"

func Copy(path, namespace, destination string, source bool) error {
	pathInContainer := JarPathInContainer
	if source {
		pathInContainer = SourcePathInContainer
	}
	if err := runKubectl([]string{"cp", path, fmt.Sprintf("%s:%s", destination, pathInContainer), "-n", namespace}...); err != nil {
		return err
	}
	return nil
}

func IsJarPresent(podName string) bool {
	return runKubectl([]string{"exec", podName, "--", "ls", JarPathInContainer}...) == nil
}

func Logs(podName string) error {
	command, interceptor := configureKubectlCmd("logs", "--since=10s", podName)
	command.Stdout = os.Stdout // so that we can print the output of the command
	return runKubectlCmd(command, interceptor)
}

func Apply(path, namespace string) error {
	return runKubectl([]string{"apply", "-f", path, "-n", namespace}...)
}

func runKubectl(args ...string) error {
	command, interceptor := configureKubectlCmd(args...)
	return runKubectlCmd(command, interceptor)
}

func runKubectlCmd(command *exec.Cmd, interceptor *log.ErrorInterceptor) error {
	err := command.Run()
	if err != nil {
		if len(interceptor.ErrorMsg) > 0 {
			return fmt.Errorf("%v: %s", err, interceptor.ErrorMsg)
		}
		return err
	}
	return nil
}

func configureKubectlCmd(args ...string) (*exec.Cmd, *log.ErrorInterceptor) {
	command := exec.Command(kubectl, args...)
	interceptor := log.GetErrorInterceptor()
	command.Stderr = interceptor
	return command, interceptor
}

func GetK8SClientFlavor() string {
	return kubectl
}

func init() {
	// first check if oc is present
	_, err := exec.LookPath("oc")
	if err != nil {
		// if oc is not present, check if kubectl is
		_, err = exec.LookPath("kubectl")
		if err != nil {
			log.Error(fmt.Errorf("neither oc or kubectl were found in the path, aborting"))
			os.Exit(1)
		}
		kubectl = "kubectl"
		return
	}
	kubectl = "oc"
}
