package k8s

import (
	"archive/tar"
	"fmt"
	"github.com/gobwas/glob"
	servicecatalogclienset "github.com/kubernetes-incubator/service-catalog/pkg/client/clientset_generated/clientset/typed/servicecatalog/v1beta1"
	"github.com/pkg/errors"
	devexp "github.com/snowdrop/component-operator/pkg/apis/clientset/versioned/typed/component/v1alpha2"
	"github.com/snowdrop/component-operator/pkg/apis/component/v1alpha2"
	io2 "github.com/snowdrop/kreate/pkg/io"
	log2 "github.com/snowdrop/kreate/pkg/log"
	"github.com/snowdrop/kreate/pkg/validation"
	"io"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"os"
	"path"
	"path/filepath"
	"time"
)

const (
	// waitForPodTimeOut controls how long we should wait for a pod before giving up
	waitForPodTimeOut = 240 * time.Second
)

type Client struct {
	KubeClient           kubernetes.Interface
	ServiceCatalogClient *servicecatalogclienset.ServicecatalogV1beta1Client
	DevexpClient         devexp.DevexpV1alpha2Interface
	KubeConfig           clientcmd.ClientConfig
	Namespace            string
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

		serviceCatalogClient, err := servicecatalogclienset.NewForConfig(config)
		io2.LogErrorAndExit(err, "error creating k8s service catalog client")
		client.ServiceCatalogClient = serviceCatalogClient

		client.DevexpClient, err = devexp.NewForConfig(config)
		io2.LogErrorAndExit(err, "error creating devexp client")

		namespace, _, err := client.KubeConfig.Namespace()
		io2.LogErrorAndExit(err, "error retrieving namespace")
		client.Namespace = namespace
	}

	return client
}

// CopyFile copies localPath directory or list of files in copyFiles list to the directory in running Pod.
// copyFiles is list of changed files captured during `odo watch` as well as binary file path
// During copying binary components, localPath represent base directory path to binary and copyFiles contains path of binary
// During copying local source components, localPath represent base directory path whereas copyFiles is empty
// During `odo watch`, localPath represent base directory path whereas copyFiles contains list of changed Files
func (c *Client) CopyFile(localPath string, targetPodName string, targetPath string, copyFiles []string, globExps []string) error {
	dest := path.Join(targetPath, filepath.Base(localPath))
	reader, writer := io.Pipe()
	// inspired from https://github.com/kubernetes/kubernetes/blob/master/pkg/kubectl/cmd/cp.go#L235
	go func() {
		defer writer.Close()

		var err error
		err = makeTar(localPath, dest, writer, copyFiles, globExps)
		io2.LogErrorAndExit(err, "couldn't tar local files to send to cluster")

	}()

	// cmdArr will run inside container
	cmdArr := []string{"tar", "xf", "-", "-C", targetPath, "--strip", "1"}
	err := c.ExecCMDInContainer(targetPodName, cmdArr, nil, nil, reader, false)
	if err != nil {
		return err
	}
	return nil
}

// makeTar function is copied from https://github.com/kubernetes/kubernetes/blob/master/pkg/kubectl/cmd/cp.go#L309
// srcPath is ignored if files is set
func makeTar(srcPath, destPath string, writer io.Writer, files []string, globExps []string) error {
	// TODO: use compression here?
	tarWriter := tar.NewWriter(writer)
	defer tarWriter.Close()
	srcPath = path.Clean(srcPath)
	destPath = path.Clean(destPath)

	if len(files) != 0 {
		//watchTar
		for _, fileName := range files {
			if validation.CheckFileExist(fileName) {
				// Fetch path of source file relative to that of source base path so that it can be passed to recursiveTar
				// which uses path relative to base path for taro header to correctly identify file location when untarred
				srcFile, err := filepath.Rel(srcPath, fileName)
				if err != nil {
					return err
				}
				srcFile = filepath.Join(filepath.Base(srcPath), srcFile)
				// The file could be a regular file or even a folder, so use recursiveTar which handles symlinks, regular files and folders
				err = recursiveTar(path.Dir(srcPath), srcFile, path.Dir(destPath), srcFile, tarWriter, globExps)
				if err != nil {
					return err
				}
			}
		}
	} else {
		return recursiveTar(path.Dir(srcPath), path.Base(srcPath), path.Dir(destPath), path.Base(destPath), tarWriter, globExps)
	}

	return nil
}

// recursiveTar function is copied from https://github.com/kubernetes/kubernetes/blob/master/pkg/kubectl/cmd/cp.go#L319
func recursiveTar(srcBase, srcFile, destBase, destFile string, tw *tar.Writer, globExps []string) error {
	joinedPath := path.Join(srcBase, srcFile)
	matchedPathsDir, err := filepath.Glob(joinedPath)
	if err != nil {
		return err
	}

	// checking the files which are allowed by glob matching
	matchedPaths := make([]string, 0, len(matchedPathsDir))
	for _, p := range matchedPathsDir {
		matched, err := IsGlobExpMatch(p, globExps)
		if err != nil {
			return err
		}
		if !matched {
			matchedPaths = append(matchedPaths, p)
		}
	}

	// adding the files for taring
	for _, matchedPath := range matchedPaths {
		stat, err := os.Lstat(matchedPath)
		if err != nil {
			return err
		}
		if stat.IsDir() {
			files, err := ioutil.ReadDir(matchedPath)
			if err != nil {
				return err
			}
			if len(files) == 0 {
				//case empty directory
				hdr, _ := tar.FileInfoHeader(stat, matchedPath)
				hdr.Name = destFile
				if err := tw.WriteHeader(hdr); err != nil {
					return err
				}
			}
			for _, f := range files {
				if err := recursiveTar(srcBase, path.Join(srcFile, f.Name()), destBase, path.Join(destFile, f.Name()), tw, globExps); err != nil {
					return err
				}
			}
			return nil
		} else if stat.Mode()&os.ModeSymlink != 0 {
			//case soft link
			hdr, _ := tar.FileInfoHeader(stat, joinedPath)
			target, err := os.Readlink(joinedPath)
			if err != nil {
				return err
			}

			hdr.Linkname = target
			hdr.Name = destFile
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
		} else {
			//case regular file or other file type like pipe
			hdr, err := tar.FileInfoHeader(stat, joinedPath)
			if err != nil {
				return err
			}
			hdr.Name = destFile

			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}

			f, err := os.Open(joinedPath)
			if err != nil {
				return err
			}
			defer f.Close()

			if _, err := io.Copy(tw, f); err != nil {
				return err
			}
			return f.Close()
		}
	}
	return nil
}

// IsGlobExpMatch compiles strToMatch against each of the passed globExps
// Parameters:
// strToMatch : a string for matching against the rules
// globExps : a list of glob patterns to match strToMatch with
// Returns: true if there is any match else false the error (if any)
func IsGlobExpMatch(strToMatch string, globExps []string) (bool, error) {
	for _, globExp := range globExps {
		pattern, err := glob.Compile(globExp)
		if err != nil {
			return false, err
		}
		matched := pattern.Match(strToMatch)
		if matched {
			return true, nil
		}
	}
	return false, nil
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

func (c *Client) WaitForComponent(name string, desiredPhase v1alpha2.ComponentPhase, waitMessage string) (*v1alpha2.Component, error) {
	s := log2.Spinner(waitMessage)
	defer s.End(false)

	/*w, err := c.KubeClient.CoreV1().RESTClient().Get().
		Namespace(c.Namespace).
		Resource("components." +v1alpha2.GroupName).
		Name(name).
		VersionedParams(&metav1.ListOptions{
			Watch: true,
		}, scheme.ParameterCodec).
		Timeout(30 * time.Second).
		Watch()
	if err != nil {
		return nil, errors.Wrapf(err, "unable to watch for component %s", name)
	}*/
	//query := Query(c.Namespace, name)
	var timeout int64 = 10
	w, err := c.DevexpClient.
		Components(c.Namespace).
		Watch(metav1.ListOptions{
			TimeoutSeconds: &timeout,
			FieldSelector:  fields.OneTermEqualSelector("metadata.name", name).String(),
		})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to watch for component %s", name)
	}
	defer w.Stop()

	podChannel := make(chan *v1alpha2.Component)
	watchErrorChannel := make(chan error)

	go func() {
	loop:
		for {
			val, ok := <-w.ResultChan()
			object := val.Object
			fmt.Printf("received %#v", object)
			if !ok {
				watchErrorChannel <- errors.New("watch channel was closed")
				break loop
			}

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
			if e, ok := object.(*v1alpha2.Component); ok {
				switch e.Status.Phase {
				case desiredPhase:
					s.End(true)
					podChannel <- e
					break loop
				case v1alpha2.ComponentFailed, v1alpha2.ComponentUnknown:
					watchErrorChannel <- errors.Errorf("component %s status %s", e.Name, e.Status.Phase)
					break loop
				}
			} else {
				watchErrorChannel <- errors.New("unable to convert event object to Component")
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
	case <-time.After(waitForPodTimeOut):
		return nil, errors.Errorf("waited %s but couldn't find running component named '%s'", waitForPodTimeOut, name)
	}
}

// WaitAndGetPod block and waits until pod matching selector is in in Running state
// desiredPhase cannot be PodFailed or PodUnknown
func (c *Client) WaitAndGetPod(selector string, desiredPhase corev1.PodPhase, waitMessage string) (*corev1.Pod, error) {
	s := log2.Spinner(waitMessage)
	defer s.End(false)

	/*w, err := c.KubeClient.CoreV1().RESTClient().Get().
	Namespace(c.Namespace).
	Resource("pods").
	VersionedParams(&metav1.ListOptions{
		Watch:         true,
		LabelSelector: selector,
	}, scheme.ParameterCodec).
	Timeout(30 * time.Second).
	Watch()*/

	w, err := c.KubeClient.CoreV1().Pods(c.Namespace).Watch(metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to watch pod")
	}
	defer w.Stop()

	podChannel := make(chan *corev1.Pod)
	watchErrorChannel := make(chan error)

	go func() {
	loop:
		for {
			val, ok := <-w.ResultChan()
			if !ok {
				watchErrorChannel <- errors.New("watch channel was closed")
				break loop
			}
			if e, ok := val.Object.(*corev1.Pod); ok {
				switch e.Status.Phase {
				case desiredPhase:
					if desiredPhase == corev1.PodRunning {
						conditions := e.Status.Conditions
						if len(conditions) > 0 {
							for _, condition := range conditions {
								if condition.Type == corev1.ContainersReady && condition.Status == corev1.ConditionTrue {
									s.End(true)
									podChannel <- e
									break loop
								}
							}
						}
					} else {
						s.End(true)
						podChannel <- e
						break loop
					}
				case corev1.PodFailed, corev1.PodUnknown:
					watchErrorChannel <- errors.Errorf("pod %s status %s", e.Name, e.Status.Phase)
					break loop
				}
			} else {
				watchErrorChannel <- errors.New("unable to convert event object to Pod")
				break loop
			}
		}
		close(podChannel)
		close(watchErrorChannel)
	}()

	select {
	case val := <-podChannel:
		time.Sleep(20 * time.Second) // wait for 5 seconds before returning to let pod finish up
		return val, nil
	case err := <-watchErrorChannel:
		return nil, err
	case <-time.After(waitForPodTimeOut):
		return nil, errors.Errorf("waited %s but couldn't find running pod matching selector: '%s'", waitForPodTimeOut, selector)
	}
}
