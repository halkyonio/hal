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

func Apply(path, namespace string) error {
	return runKubectl([]string{"apply", "-f", path, "-n", namespace}...)
}

func runKubectl(args ...string) error {
	command := exec.Command(kubectl, args...)
	interceptor := log.GetErrorInterceptor()
	command.Stderr = interceptor
	err := command.Run()
	if err != nil {
		if len(interceptor.ErrorMsg) > 0 {
			return fmt.Errorf("%v: %s", err, interceptor.ErrorMsg)
		}
		return err
	}
	return nil
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
