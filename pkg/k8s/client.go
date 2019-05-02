package k8s

import (
	servicecatalogclienset "github.com/kubernetes-incubator/service-catalog/pkg/client/clientset_generated/clientset/typed/servicecatalog/v1beta1"
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
func GetClient() (*Client, error) {
	if client == nil {
		// initialize client-go clients
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		configOverrides := &clientcmd.ConfigOverrides{}
		client.KubeConfig = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
		config, err := client.KubeConfig.ClientConfig()
		if err != nil {
			return nil, err
		}
		kubeClient, err := kubernetes.NewForConfig(config)
		if err != nil {
			return nil, err
		}
		client.KubeClient = kubeClient
		serviceCatalogClient, err := servicecatalogclienset.NewForConfig(config)
		if err != nil {
			return nil, err
		}
		client.ServiceCatalogClient = serviceCatalogClient
		namespace, _, err := client.KubeConfig.Namespace()
		if err != nil {
			return nil, err
		}
		client.Namespace = namespace
	}

	return client, nil
}
