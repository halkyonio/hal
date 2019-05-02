package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"github.com/ghodss/yaml"
	scv1beta1 "github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/v1beta1"
	servicecatalogclienset "github.com/kubernetes-incubator/service-catalog/pkg/client/clientset_generated/clientset/typed/servicecatalog/v1beta1"
	log "github.com/sirupsen/logrus"
	"github.com/snowdrop/odo-scaffold-plugin/pkg/scaffold"
	"github.com/snowdrop/odo-scaffold-plugin/pkg/ui"
	"github.com/spf13/cobra"
	"io"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

const (
	ServiceEndpoint          = "https://generator.snowdrop.me"
	ReleaseSuffix            = ".RELEASE"
	serviceCatalogAnnotation = `@ServiceCatalog(instances = @ServiceCatalogInstance(
        name = "{{.Name}}",
        serviceClass = "{{.Class}}",
        servicePlan = "{{.Plan}}",
        parameters = {
{{parameters .Parameters}}
        },
        bindingSecret = "my-db-secret")
)

`
)

func main() {
	p := &scaffold.Project{}

	createCmd := &cobra.Command{
		Use:   "scaffold [flags]",
		Short: "Create a Spring Boot maven project",
		Long:  `Create a Spring Boot maven project.`,
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// fail fast if needed
			useTemplate := len(p.Template) > 0
			useModules := len(p.Modules) > 0
			if useTemplate && useModules {
				return fmt.Errorf("specifying both modules and template is not currently supported")
			}

			c := getGeneratorServiceConfig(p.UrlService)

			// first select Spring Boot version
			versions, defaultVersion := c.GetBOMMap()
			hasSB := len(p.SpringBootVersion) > 0

			// modify given SB version if needed since we allow 2.1.3 instead of full 2.1.3.RELEASE
			if hasSB && !strings.HasSuffix(p.SpringBootVersion, ReleaseSuffix) {
				p.SpringBootVersion = p.SpringBootVersion + ReleaseSuffix
			}

			// if the user didn't specify an SB version, ask for it
			if !hasSB {
				p.SpringBootVersion = ui.Select("Spring Boot version", scaffold.GetSpringBootVersions(versions), defaultVersion)
			}

			// check that the given SB version yields a known BOM, if not ask the user for a supported SB version
			bom, ok := versions[p.SpringBootVersion]
			if !ok {
				s := ui.ErrorMessage("Unknown Spring Boot version", p.SpringBootVersion)
				p.SpringBootVersion = ui.Select(s, scaffold.GetSpringBootVersions(versions), defaultVersion)
			} else if hasSB {
				// if we provided an SB version and it yields a valid BOM, display it
				ui.OutputSelection("Selected Spring Boot", p.SpringBootVersion)
			}

			p.SnowdropBomVersion = bom.Snowdrop
			if len(bom.Supported) > 0 {
				if !cmd.Flag("supported").Changed {
					p.UseSupported = ui.Proceed(fmt.Sprintf("Use %s supported version", p.SpringBootVersion))
				}

				if p.UseSupported {
					p.SnowdropBomVersion = c.GetSupportedVersionFor(p.SpringBootVersion)
					ui.OutputSelection("Selected supported Spring Boot", p.SnowdropBomVersion)
				}
			}

			// deal with template
			templateNames := c.GetTemplateNames()
			if useTemplate {
				if !isContained(p.Template, templateNames) {
					// provided template doesn't exist, select one from available
					p.Template = ui.Select(ui.ErrorMessage("Unknown template", p.Template), templateNames)
				} else {
					ui.OutputSelection("Selected template", p.Template)
				}
			}

			// deal with modules
			if useModules {
				// check if all provided modules are known
				moduleNames := getCompatibleModuleNamesFor(p)
				sort.Strings(moduleNames)
				unknown := make([]string, 0, len(moduleNames))
				valid := make([]string, 0, len(moduleNames))
				for _, module := range p.Modules {
					if !isContained(module, moduleNames) {
						unknown = append(unknown, module)
					} else {
						valid = append(valid, module)
					}
				}

				if !isContained("core", valid) {
					valid = append(valid, "core")
				}
				ui.OutputSelection("Selected modules", strings.Join(valid, ","))

				if len(unknown) > 0 {
					p.Modules = ui.MultiSelect(ui.ErrorMessage("Unknown modules", strings.Join(unknown, ",")), moduleNames, valid)
				}
			}

			// if user didn't specify either template or modules, ask what to do
			if !useModules && !useTemplate {
				if ui.Proceed("Create from template") {
					p.Template = ui.Select("Available templates", templateNames)
					useTemplate = true
				} else {
					p.Modules = ui.MultiSelect("Select modules", getCompatibleModuleNamesFor(p), []string{"core"})
					useModules = true
				}
			}

			// if we're using a template, ask additional information
			if useTemplate {
				// only ask about ap4k if the user didn't specify the flag
				if !cmd.Flag("ap4k").Changed {
					p.UseAp4k = ui.Proceed("Use ap4k to generate OpenShift / Kubernetes resources")
				}

				if p.UseAp4k && ui.Proceed("Create a service from service catalog") {
					generateAp4kAnnotations()
				}
			}

			p.GroupId = ui.Ask("Group Id", p.GroupId, "me.snowdrop")
			p.ArtifactId = ui.Ask("Artifact Id", p.ArtifactId, "myproject")
			p.Version = ui.Ask("Version", p.Version, "1.0.0-SNAPSHOT")
			p.PackageName = ui.Ask("Package name", p.PackageName, p.GroupId+"."+p.ArtifactId)

			currentDir, _ := os.Getwd()
			p.OutDir = ui.Ask(fmt.Sprintf("Project location (immediate child directory of %s)", currentDir), p.OutDir)

			client := http.Client{}

			form := url.Values{}
			form.Add("template", p.Template)
			form.Add("groupid", p.GroupId)
			form.Add("artifactid", p.ArtifactId)
			form.Add("version", p.Version)
			form.Add("packagename", p.PackageName)
			form.Add("snowdropbom", p.SnowdropBomVersion)
			form.Add("springbootversion", p.SpringBootVersion)
			form.Add("outdir", p.OutDir)
			form.Add("ap4k", strconv.FormatBool(p.UseAp4k))
			for _, v := range p.Modules {
				if v != "" {
					form.Add("module", v)
				}
			}

			parameters := form.Encode()
			if parameters != "" {
				parameters = "?" + parameters
			}

			u := strings.Join([]string{p.UrlService, "app"}, "/") + parameters
			log.Infof("URL of the request calling the service is %s", u)
			req, err := http.NewRequest(http.MethodGet, u, strings.NewReader(""))

			if err != nil {
				return err
			}
			addClientHeader(req)

			res, err := client.Do(req)
			if err != nil {
				return err
			}
			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				return err
			}

			dir := filepath.Join(currentDir, p.OutDir)
			zipFile := dir + ".zip"

			err = ioutil.WriteFile(zipFile, body, 0644)
			if err != nil {
				return fmt.Errorf("failed to download file %s due to %s", zipFile, err)
			}
			err = Unzip(zipFile, dir)
			if err != nil {
				return fmt.Errorf("failed to unzip new project file %s due to %s", zipFile, err)
			}
			err = os.Remove(zipFile)
			if err != nil {
				return err
			}
			return nil
		},
	}

	createCmd.Flags().StringVarP(&p.Template, "template", "t", "", "Template name used to select the project to be created")
	createCmd.Flags().StringVarP(&p.UrlService, "urlservice", "u", ServiceEndpoint, "URL of the HTTP Server exposing the spring boot service")
	createCmd.Flags().StringSliceVarP(&p.Modules, "module", "m", []string{}, "Spring Boot modules/starters")
	createCmd.Flags().StringVarP(&p.GroupId, "groupid", "g", "", "GroupId : com.example")
	createCmd.Flags().StringVarP(&p.ArtifactId, "artifactid", "i", "", "ArtifactId: demo")
	createCmd.Flags().StringVarP(&p.Version, "version", "v", "", "Version: 0.0.1-SNAPSHOT")
	createCmd.Flags().StringVarP(&p.PackageName, "packagename", "p", "", "Package Name: com.example.demo")
	createCmd.Flags().StringVarP(&p.SpringBootVersion, "springbootversion", "s", "", "Spring Boot Version")
	createCmd.Flags().BoolVarP(&p.UseAp4k, "ap4k", "a", false, "Use ap4k when possible")
	createCmd.Flags().BoolVarP(&p.UseSupported, "supported", "o", false, "Use supported version")

	err := createCmd.Execute()
	if err != nil {
		fmt.Print(err.Error())
	}
}

type svcInstance struct {
	Class      string
	Plan       string
	Parameters map[string]string
	Name       string
}

func generateAp4kAnnotations() error {
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
	log.Infof("ap4k annotation:\n%s", strings.TrimSpace(tpl.String()))

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

func getYamlFrom(url, endpoint string, result interface{}) {
	// Call the /config endpoint to get the configuration
	URL := strings.Join([]string{url, endpoint}, "/")
	client := http.Client{}

	req, err := http.NewRequest(http.MethodGet, URL, strings.NewReader(""))

	if err != nil {
		log.Error(err.Error())
	}
	addClientHeader(req)

	res, err := client.Do(req)
	if err != nil {
		log.Error(err.Error())
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Error(err.Error())
	}

	if strings.Contains(string(body), "Application is not available") {
		log.Fatal("Generator service is not available")
	}

	err = yaml.Unmarshal(body, &result)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func getGeneratorServiceConfig(url string) *scaffold.Config {
	c := &scaffold.Config{}
	getYamlFrom(url, "config", c)

	return c
}

func getCompatibleModuleNamesFor(p *scaffold.Project) []string {
	modules := &[]scaffold.Module{}
	getYamlFrom(p.UrlService, "modules/"+p.SpringBootVersion, modules)
	return scaffold.GetModuleNamesFor(*modules)
}

func addClientHeader(req *http.Request) {
	userAgent := "snowdrop-scaffold/1.0"
	req.Header.Set("User-Agent", userAgent)
}

func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		name := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			err := os.MkdirAll(name, os.ModePerm)
			if err != nil {
				return err
			}
		} else {
			var fdir string
			if lastIndex := strings.LastIndex(name, string(os.PathSeparator)); lastIndex > -1 {
				fdir = name[:lastIndex]
			}

			err = os.MkdirAll(fdir, os.ModePerm)
			if err != nil {
				return err
			}
			f, err := os.OpenFile(
				name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func isContained(element string, sortedElements []string) bool {
	i := sort.SearchStrings(sortedElements, element)
	if i < len(sortedElements) && sortedElements[i] == element {
		return true
	}
	return false
}
