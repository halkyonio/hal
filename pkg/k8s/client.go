package k8s

import (
	servicecatalogclienset "github.com/kubernetes-incubator/service-catalog/pkg/client/clientset_generated/clientset/typed/servicecatalog/v1beta1"
	"github.com/snowdrop/kreate/pkg/io"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
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
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		configOverrides := &clientcmd.ConfigOverrides{}
		client.KubeConfig = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
		config, err := client.KubeConfig.ClientConfig()
		io.LogErrorAndExit(err, "error creating k8s config")

		kubeClient, err := kubernetes.NewForConfig(config)
		io.LogErrorAndExit(err, "error creating k8s client")
		client.KubeClient = kubeClient

		serviceCatalogClient, err := servicecatalogclienset.NewForConfig(config)
		io.LogErrorAndExit(err, "error creating k8s service catalog client")
		client.ServiceCatalogClient = serviceCatalogClient

		namespace, _, err := client.KubeConfig.Namespace()
		io.LogErrorAndExit(err, "error retrieving namespace")
		client.Namespace = namespace
	}

	return client
}
