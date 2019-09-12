package component

import (
	"fmt"
	"github.com/spf13/cobra"
	"halkyon.io/hal/pkg/cmdutil"
	"halkyon.io/hal/pkg/io"
	"halkyon.io/hal/pkg/scaffold"
	"halkyon.io/hal/pkg/ui"
	"halkyon.io/hal/pkg/validation"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	ktemplates "k8s.io/kubectl/pkg/util/templates"
)

const (
	serviceEndpoint    = "https://generator.snowdrop.me"
	releaseSuffix      = ".RELEASE"
	projectCommandName = "spring-boot"
)

var (
	projectExample = ktemplates.Examples(`  # Creates a Spring Boot maven project project using the REST HTTP client template
  %[1]s  \
            -i client-sb \
            -g me.myspringboot \
            -p me.myspringboot.demo \
            -s 2.1.6.RELEASE \
            -t client \
            -v 1.0.0-SNAPSHOT \
            --supported=false  \
           client-sb`)
)

func NewCmdProject(parent string) *cobra.Command {
	p := &project{}

	currentDir, _ := os.Getwd()
	createCmd := &cobra.Command{
		Use:   fmt.Sprintf("%s [flags] <project location (immediate child directory of %s)>", projectCommandName, currentDir),
		Short: "Create a Spring Boot maven project",
		Long:  `Create a Spring Boot maven project.`,
		Args:  cobra.ExactArgs(1),
		Example: fmt.Sprintf(projectExample, "hal component spring-boot"),
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.GenericRun(p, cmd, args)
		},
	}

	createCmd.Flags().StringVarP(&p.Template, "template", "t", "", "Template name used to select the project to be created")
	createCmd.Flags().StringVarP(&p.UrlService, "urlservice", "u", serviceEndpoint, "URL of the HTTP Server exposing the spring boot service")
	createCmd.Flags().StringSliceVarP(&p.Modules, "module", "m", []string{}, "Comma-separated list of Spring Boot modules/starters to add to the project")
	createCmd.Flags().StringVarP(&p.GroupId, "groupid", "g", "", "Maven group id e.g. com.example")
	createCmd.Flags().StringVarP(&p.ArtifactId, "artifactid", "i", "", "Maven artifact id e.g. demo")
	createCmd.Flags().StringVarP(&p.Version, "version", "v", "", "Maven version e.g. 0.0.1-SNAPSHOT")
	createCmd.Flags().StringVarP(&p.PackageName, "packagename", "p", "", "Package name (defaults to <group id>.<artifact id>)")
	createCmd.Flags().StringVarP(&p.SpringBootVersion, "springbootversion", "s", "", "Spring Boot Version")
	createCmd.Flags().BoolVarP(&p.UseSupported, "supported", "o", false, "Use Snowdrop supported version of Spring Boot")

	return createCmd
}

func getGeneratorServiceConfig(url string) *scaffold.Config {
	c := &scaffold.Config{}
	io.UnmarshallYamlFromHttp(url, "config", c)

	return c
}

func getCompatibleModuleNamesFor(p *project) []string {
	modules := &[]scaffold.Module{}
	io.UnmarshallYamlFromHttp(p.UrlService, "modules/"+p.SpringBootVersion, modules)
	return scaffold.GetModuleNamesFor(*modules)
}

func isContained(element string, sortedElements []string) bool {
	i := sort.SearchStrings(sortedElements, element)
	if i < len(sortedElements) && sortedElements[i] == element {
		return true
	}
	return false
}

func (p *project) Complete(name string, cmd *cobra.Command, args []string) error {
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
	if hasSB && !strings.HasSuffix(p.SpringBootVersion, releaseSuffix) {
		p.SpringBootVersion = p.SpringBootVersion + releaseSuffix
	}

	// if the user didn't specify an SB version, ask for it
	if !hasSB {
		p.SpringBootVersion = ui.Select("Spring Boot version", scaffold.GetSpringBootVersions(versions), defaultVersion)
	}

	// check that the given SB version yields a known BOM, if not ask the user for a supported SB version
	bom, ok := versions[p.SpringBootVersion]
	if !ok {
		s := ui.SelectFromOtherErrorMessage("Unknown Spring Boot version", p.SpringBootVersion)
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
			p.Template = ui.Select(ui.SelectFromOtherErrorMessage("Unknown template", p.Template), templateNames)
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
		if !isContained("halkyon", valid) {
			valid = append(valid, "halkyon")
		}
		ui.OutputSelection("Selected modules", strings.Join(valid, ","))

		if len(unknown) > 0 {
			p.Modules = ui.MultiSelect(ui.SelectFromOtherErrorMessage("Unknown modules", strings.Join(unknown, ",")), moduleNames, valid)
		}
	}

	// if user didn't specify either template or modules, ask what to do
	if !useModules && !useTemplate {
		if ui.Proceed("Create from template") {
			p.Template = ui.Select("Available templates", templateNames)
			useTemplate = true
		} else {
			p.Modules = ui.MultiSelect("Select modules", getCompatibleModuleNamesFor(p), []string{"core", "halkyon"})
			useModules = true
		}
	}

	p.GroupId = ui.Ask("Group Id", p.GroupId, "dev.snowdrop")
	p.ArtifactId = ui.Ask("Artifact Id", p.ArtifactId, "myproject")
	p.Version = ui.Ask("Version", p.Version, "1.0.0-SNAPSHOT")
	p.PackageName = ui.Ask("Package name", p.PackageName, p.GroupId+"."+p.ArtifactId)

	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}
	p.fileName = args[0]
	p.zipFile = filepath.Join(currentDir, p.fileName+".zip")

	return nil
}

func (p *project) Validate() error {
	if validation.CheckFileExist(p.fileName) {
		currentDir, _ := os.Getwd()
		return fmt.Errorf("a file named %s already exists in %s", p.fileName, currentDir)
	}
	return nil
}
func (p *project) Run() error {
	form := &url.Values{}
	form.Add("template", p.Template)
	form.Add("groupid", p.GroupId)
	form.Add("artifactid", p.ArtifactId)
	form.Add("version", p.Version)
	form.Add("packagename", p.PackageName)
	form.Add("snowdropbom", p.SnowdropBomVersion)
	form.Add("springbootversion", p.SpringBootVersion)
	form.Add("outdir", p.fileName)
	for _, v := range p.Modules {
		if v != "" {
			form.Add("module", v)
		}
	}

	body := io.HttpGet(p.UrlService, "app", form)

	err := ioutil.WriteFile(p.zipFile, body, 0644)
	if err != nil {
		return fmt.Errorf("failed to download file %s due to %s", p.zipFile, err)
	}
	// output zipped file into proper child directory
	dir := filepath.Join(filepath.Dir(p.zipFile), p.fileName)
	err = io.Unzip(p.zipFile, dir)
	if err != nil {
		return fmt.Errorf("failed to unzip new project file %s due to %s", p.zipFile, err)
	}
	err = os.Remove(p.zipFile)
	if err != nil {
		return err
	}
	return nil
}

type project struct {
	GroupId     string
	ArtifactId  string
	Version     string
	PackageName string
	zipFile     string
	fileName    string
	Template    string `yaml:"template"  json:"template"`

	SnowdropBomVersion string
	SpringBootVersion  string
	Modules            []string

	UrlService   string
	UseSupported bool
}
