package k8s

import (
	"archive/tar"
	servicecatalogclienset "github.com/kubernetes-incubator/service-catalog/pkg/client/clientset_generated/clientset/typed/servicecatalog/v1beta1"
	"github.com/pkg/errors"
	io2 "github.com/snowdrop/kreate/pkg/io"
	"io"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"os"
	"path"
	"path/filepath"
)

type Client struct {
	KubeClient           kubernetes.Interface
	ServiceCatalogClient *servicecatalogclienset.ServicecatalogV1beta1Client
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

		namespace, _, err := client.KubeConfig.Namespace()
		io2.LogErrorAndExit(err, "error retrieving namespace")
		client.Namespace = namespace
	}

	return client
}

func (c *Client) CopyFile(localPath string, targetPodName string, targetPath string) error {
	dest := path.Join(targetPath, filepath.Base(localPath))
	reader, writer := io.Pipe()
	// inspired from https://github.com/kubernetes/kubernetes/blob/master/pkg/kubectl/cmd/cp/cp.go
	go func() {
		defer writer.Close()

		var err error
		err = makeTar(localPath, dest, writer)
		io2.LogErrorAndExit(err, "couldn't create tar")

	}()

	// cmdArr will run inside container
	cmdArr := []string{"tar", "xf", "-", "-C", targetPath, "--strip", "1"}
	err := c.ExecCMDInContainer(targetPodName, cmdArr, nil, nil, reader, false)
	if err != nil {
		return err
	}
	return nil
}

// makeTar function is copied from https://github.com/kubernetes/kubernetes/blob/master/pkg/kubectl/cmd/cp/cp.go
// srcPath is ignored if files is set
func makeTar(srcPath, destPath string, writer io.Writer) error {
	// TODO: use compression here?
	tarWriter := tar.NewWriter(writer)
	defer tarWriter.Close()
	srcPath = path.Clean(srcPath)
	destPath = path.Clean(destPath)

	return recursiveTar(path.Dir(srcPath), path.Base(srcPath), path.Dir(destPath), path.Base(destPath), tarWriter)
}

// recursiveTar function is copied from https://github.com/kubernetes/kubernetes/blob/master/pkg/kubectl/cmd/cp/cp.go
func recursiveTar(srcBase, srcFile, destBase, destFile string, tw *tar.Writer) error {
	joinedPath := path.Join(srcBase, srcFile)
	matchedPathsDir, err := filepath.Glob(joinedPath)
	if err != nil {
		return err
	}

	// adding the files for taring
	for _, matchedPath := range matchedPathsDir {
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
				if err := recursiveTar(srcBase, path.Join(srcFile, f.Name()), destBase, path.Join(destFile, f.Name()), tw); err != nil {
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
