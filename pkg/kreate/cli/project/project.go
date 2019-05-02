package project

import (
	"fmt"
	"github.com/snowdrop/kreate/pkg/io"
	"github.com/snowdrop/kreate/pkg/scaffold"
	"github.com/snowdrop/kreate/pkg/servicecatalog"
	"github.com/snowdrop/kreate/pkg/ui"
	"github.com/spf13/cobra"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	ServiceEndpoint = "https://generator.snowdrop.me"
	ReleaseSuffix   = ".RELEASE"
)

func NewCmdProject() *cobra.Command {
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
					err := servicecatalog.GenerateAp4kAnnotations()
					io.LogErrorAndExit(err, "error generating ap4k annotations")
				}
			}

			p.GroupId = ui.Ask("Group Id", p.GroupId, "me.snowdrop")
			p.ArtifactId = ui.Ask("Artifact Id", p.ArtifactId, "myproject")
			p.Version = ui.Ask("Version", p.Version, "1.0.0-SNAPSHOT")
			p.PackageName = ui.Ask("Package name", p.PackageName, p.GroupId+"."+p.ArtifactId)

			currentDir, _ := os.Getwd()
			p.OutDir = ui.Ask(fmt.Sprintf("Project location (immediate child directory of %s)", currentDir), p.OutDir)

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

			body := io.HttpGet(p.UrlService, "app")

			dir := filepath.Join(currentDir, p.OutDir)
			zipFile := dir + ".zip"

			err := ioutil.WriteFile(zipFile, body, 0644)
			if err != nil {
				return fmt.Errorf("failed to download file %s due to %s", zipFile, err)
			}
			err = io.Unzip(zipFile, dir)
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

	return createCmd
}

func getGeneratorServiceConfig(url string) *scaffold.Config {
	c := &scaffold.Config{}
	io.GetYamlFrom(url, "config", c)

	return c
}

func getCompatibleModuleNamesFor(p *scaffold.Project) []string {
	modules := &[]scaffold.Module{}
	io.GetYamlFrom(p.UrlService, "modules/"+p.SpringBootVersion, modules)
	return scaffold.GetModuleNamesFor(*modules)
}

func isContained(element string, sortedElements []string) bool {
	i := sort.SearchStrings(sortedElements, element)
	if i < len(sortedElements) && sortedElements[i] == element {
		return true
	}
	return false
}
