package servicecatalog

import (
	"bytes"
	"fmt"
	scv1beta1 "github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/v1beta1"
	servicecatalogclienset "github.com/kubernetes-incubator/service-catalog/pkg/client/clientset_generated/clientset/typed/servicecatalog/v1beta1"
	"github.com/sirupsen/logrus"
	"github.com/snowdrop/kreate/pkg/servicecatalog/ui"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"strings"
	"text/template"
)

const serviceCatalogAnnotation = `@ServiceCatalog(instances = @ServiceCatalogInstance(
        name = "{{.Name}}",
        serviceClass = "{{.Class}}",
        servicePlan = "{{.Plan}}",
        parameters = {
{{parameters .Parameters}}
        },
        bindingSecret = "my-db-secret")
)

`

type svcInstance struct {
	Class      string
	Plan       string
	Parameters map[string]string
	Name       string
}

func GenerateAp4kAnnotations() error {
	classesByCategory, svcatClient, err := getServiceClassesByCategory()
	if err != nil {
		return fmt.Errorf("unable to retrieve service classes: %v", err)
	}
	if len(classesByCategory) == 0 {
		return fmt.Errorf("unable to retrieve service classes or none present")
	}
	class, serviceType := ui.SelectClassInteractively(classesByCategory)

	plans, err := GetMatchingPlans(svcatClient, class)
	if err != nil {
		return fmt.Errorf("couldn't retrieve plans for class %s: %v", class.GetExternalName(), err)
	}

	var plan string
	var svcPlan scv1beta1.ClusterServicePlan
	// if there is only one available plan, we select it
	if len(plans) == 1 {
		for k, v := range plans {
			plan = k
			svcPlan = v
		}
		//glog.V(4).Infof("Plan %s was automatically selected since it's the only one available for service %s", o.Plan, o.ServiceType)
	} else {
		// otherwise select the plan interactively
		plan = ui.SelectPlanNameInteractively(plans, "Which service plan should we use ")
		svcPlan = plans[plan]
	}

	parametersMap := ui.EnterServicePropertiesInteractively(svcPlan)
	serviceName := ui.EnterServiceNameInteractively(serviceType, "How should we name your service ")

	instance := svcInstance{
		Class:      class.GetExternalName(),
		Plan:       plan,
		Parameters: parametersMap,
		Name:       serviceName,
	}
	var tpl bytes.Buffer
	tmpl := template.New("service-create-cli")
	tmpl.Funcs(template.FuncMap{"parameters": parameters})
	t := template.Must(tmpl.Parse(serviceCatalogAnnotation))
	e := t.Execute(&tpl, instance)
	if e != nil {
		panic(e) // shouldn't happen
	}
	logrus.Infof("ap4k annotation:\n%s", strings.TrimSpace(tpl.String()))

	return nil
}

func parameters(parameters map[string]string) string {
	pAsArray := make([]string, 0, len(parameters))
	for key, value := range parameters {
		pAsArray = append(pAsArray, "\t\t@Parameter(key = \""+key+"\", value = \""+value+"\")")
	}
	return strings.Join(pAsArray, ",\n")
}

func getServiceClassesByCategory() (categories map[string][]scv1beta1.ClusterServiceClass, svcatClient *servicecatalogclienset.ServicecatalogV1beta1Client, err error) {
	// initialize client-go clients
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, nil, err
	}

	serviceCatalogClient, err := servicecatalogclienset.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}

	categories = make(map[string][]scv1beta1.ClusterServiceClass)

	classList, err := serviceCatalogClient.ClusterServiceClasses().List(metav1.ListOptions{})
	if err != nil {
		return nil, nil, err
	}
	classes := classList.Items

	// TODO: Should we replicate the classification performed in
	// https://github.com/openshift/console/blob/master/frontend/public/components/catalog/catalog-items.jsx?
	for _, class := range classes {
		tags := class.Spec.Tags
		category := "other"
		if len(tags) > 0 && len(tags[0]) > 0 {
			category = tags[0]
		}
		categories[category] = append(categories[category], class)
	}

	return categories, serviceCatalogClient, err
}

// GetMatchingPlans retrieves a map associating service plan name to service plan instance associated with the specified service
// class
func GetMatchingPlans(serviceCatalogClient *servicecatalogclienset.ServicecatalogV1beta1Client, class scv1beta1.ClusterServiceClass) (plans map[string]scv1beta1.ClusterServicePlan, err error) {
	planList, err := serviceCatalogClient.ClusterServicePlans().List(metav1.ListOptions{
		FieldSelector: "spec.clusterServiceClassRef.name==" + class.Spec.ExternalID,
	})

	plans = make(map[string]scv1beta1.ClusterServicePlan)
	for _, v := range planList.Items {
		plans[v.Spec.ExternalName] = v
	}
	return plans, err
}
