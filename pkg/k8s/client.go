package k8s

import (
	"bufio"
	"fmt"
	"github.com/pkg/errors"
	capability "halkyon.io/api/capability/clientset/versioned/typed/capability/v1beta1"
	component "halkyon.io/api/component/clientset/versioned/typed/component/v1beta1"
	"halkyon.io/api/component/v1beta1"
	link "halkyon.io/api/link/clientset/versioned/typed/link/v1beta1"
	io2 "halkyon.io/hal/pkg/io"
	log2 "halkyon.io/hal/pkg/log"
	"io"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"reflect"
	"strings"
	"time"
)

const (
	timeoutDuration = 120
	// watchTimeout controls how long we should watch a resource waiting for the expected result before giving up
	watchTimeout = timeoutDuration * time.Second
)

type Client struct {
	KubeClient              kubernetes.Interface
	HalkyonComponentClient  *component.HalkyonV1beta1Client
	HalkyonLinkClient       *link.HalkyonV1beta1Client
	HalkyonCapabilityClient *capability.HalkyonV1beta1Client
	KubeConfig              clientcmd.ClientConfig
	Namespace               string
}

var client *Client

// GetClient retrieves a client
func GetClient() *Client {
	if client == nil {
		// initialize client-go clients
		client = &Client{}
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		configOverrides := &clientcmd.ConfigOverrides{}
		client.KubeConfig = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
		config, err := client.KubeConfig.ClientConfig()
		io2.LogErrorAndExit(err, "error creating k8s config")

		kubeClient, err := kubernetes.NewForConfig(config)
		io2.LogErrorAndExit(err, "error creating k8s client")
		client.KubeClient = kubeClient

		client.HalkyonComponentClient, err = component.NewForConfig(config)
		io2.LogErrorAndExit(err, "error creating halkyon component client")

		client.HalkyonLinkClient, err = link.NewForConfig(config)
		io2.LogErrorAndExit(err, "error creating halkyon link client")

		client.HalkyonCapabilityClient, err = capability.NewForConfig(config)
		io2.LogErrorAndExit(err, "error creating halkyon capability client")

		namespace, _, err := client.KubeConfig.Namespace()
		io2.LogErrorAndExit(err, "error retrieving namespace")
		client.Namespace = namespace
	}

	return client
}

func (c *Client) ExecCommand(podName string, cmd []string, statusMsg string) error {
	// use pipes to write output from ExecCMDInContainer in yellow  to 'out' io.Writer
	pipeReader, pipeWriter := io.Pipe()
	var cmdOutput string
	// This Go routine will automatically pipe the output from ExecCMDInContainer to
	// our logger.
	go func() {
		scanner := bufio.NewScanner(pipeReader)
		for scanner.Scan() {
			line := scanner.Text()
			cmdOutput += fmt.Sprintln(line)
		}
	}()

	var s *log2.Status
	if len(statusMsg) > 0 {
		s = log2.Spinner(statusMsg)
		defer s.End(false)
	}
	err := c.ExecCMDInContainer(podName, cmd, pipeWriter, pipeWriter, nil, false)
	if err != nil {
		return fmt.Errorf("cannot run '%s' cmd in '%s' pod: %s", strings.Join(cmd, " "), podName, cmdOutput)
	}
	if s != nil {
		s.End(true)
	}
	return nil
}

// ExecCMDInContainer execute command in first container of a pod
func (c *Client) ExecCMDInContainer(podName string, cmd []string, stdout io.Writer, stderr io.Writer, stdin io.Reader, tty bool) error {

	req := c.KubeClient.CoreV1().RESTClient().
		Post().
		Namespace(c.Namespace).
		Resource("pods").
		Name(podName).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Command: cmd,
			Stdin:   stdin != nil,
			Stdout:  stdout != nil,
			Stderr:  stderr != nil,
			TTY:     tty,
		}, scheme.ParameterCodec)

	config, err := c.KubeConfig.ClientConfig()
	if err != nil {
		return errors.Wrapf(err, "unable to get Kubernetes client config")
	}

	// Connect to url (constructed from req) using SPDY (HTTP/2) protocol which allows bidirectional streams.
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return errors.Wrapf(err, "unable execute command via SPDY")
	}
	// initialize the transport of the standard shell streams
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    tty,
	})
	if err != nil {
		return errors.Wrapf(err, "error while streaming command")
	}

	return nil
}

func (c *Client) WaitForComponent(name string, desiredPhase v1beta1.ComponentPhase, waitMessage string) (*v1beta1.Component, error) {
	s := log2.Spinner(waitMessage)
	defer s.End(false)

	var timeout int64 = timeoutDuration
	w, err := c.HalkyonComponentClient.
		Components(c.Namespace).
		Watch(metav1.ListOptions{
			TimeoutSeconds: &timeout,
			FieldSelector:  fields.OneTermEqualSelector("metadata.name", name).String(),
		})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to watch for component %s", name)
	}
	defer w.Stop()

	podChannel := make(chan *v1beta1.Component)
	watchErrorChannel := make(chan error)

	go func() {
	loop:
		for {
			val := <-w.ResultChan()
			object := val.Object

			if watch.Error == val.Type {
				var msg string
				if status, ok := object.(*metav1.Status); ok {
					msg = fmt.Sprintf("error: %s", status.Message)
				} else {
					msg = fmt.Sprintf("error: %#v", object)
				}
				watchErrorChannel <- errors.New(msg)
				break loop
			}
			if e, ok := object.(*v1beta1.Component); ok {
				switch e.Status.Phase {
				case desiredPhase:
					s.End(true)
					podChannel <- e
					break loop
				case v1beta1.ComponentFailed, v1beta1.ComponentUnknown:
					watchErrorChannel <- errors.Errorf("component %s status %s", e.Name, e.Status.Phase)
					break loop
				}
			} else {
				watchErrorChannel <- errors.Errorf("unable to convert event object to Component, got %v", reflect.TypeOf(object))
				break loop
			}
		}
		close(podChannel)
		close(watchErrorChannel)
	}()

	select {
	case val := <-podChannel:
		return val, nil
	case err := <-watchErrorChannel:
		return nil, err
	case <-time.After(watchTimeout):
		return nil, errors.Errorf("waited %s but couldn't find running component named '%s'", watchTimeout, name)
	}
}
