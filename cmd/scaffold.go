package main

import (
	"archive/zip"
	"fmt"
	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	"github.com/snowdrop/odo-scaffold-plugin/pkg/scaffold"
	"github.com/snowdrop/odo-scaffold-plugin/pkg/ui"
	"github.com/spf13/cobra"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	ServiceEndpoint = "https://generator.snowdrop.me"
)

func main() {
	p := &scaffold.Project{}

	createCmd := &cobra.Command{
		Use:   "scaffold [flags]",
		Short: "Create a Spring Boot maven project",
		Long:  `Create a Spring Boot maven project.`,
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := getGeneratorServiceConfig(p.UrlService)

			// first select Spring Boot version
			versions, defaultVersion := c.GetBOMMap()
			p.SpringBootVersion = ui.Select("Spring Boot version", scaffold.GetSpringBootVersions(versions), defaultVersion)
			bom := versions[p.SpringBootVersion]
			p.SnowdropBomVersion = bom.Snowdrop
			if len(bom.Supported) > 0 && ui.Proceed("Use supported version") {
				p.SnowdropBomVersion = c.GetSupportedVersionFor(p.SpringBootVersion)
			}

			if ui.Proceed("Create from template") {
				p.Template = ui.Select("Available templates", c.GetTemplateNames())
				p.UseAp4k = ui.Proceed("Use ap4k to generate OpenShift / Kubernetes resources")
			} else {
				p.Modules = ui.MultiSelect("Select modules", getCompatibleModuleNameFor(p))
			}

			p.GroupId = ui.Ask("Group Id", "me.snowdrop")
			p.ArtifactId = ui.Ask("Artifact Id", "myproject")
			p.Version = ui.Ask("Version", "1.0.0-SNAPSHOT")
			p.PackageName = ui.Ask("Package name", p.GroupId+"."+p.ArtifactId)

			currentDir, _ := os.Getwd()
			p.OutDir = ui.Ask(fmt.Sprintf("Project location (immediate child directory of %s)", currentDir))

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
	createCmd.Flags().StringArrayVarP(&p.Modules, "module", "m", []string{}, "Spring Boot modules/starters")
	createCmd.Flags().StringVarP(&p.GroupId, "groupid", "g", "", "GroupId : com.example")
	createCmd.Flags().StringVarP(&p.ArtifactId, "artifactid", "i", "", "ArtifactId: demo")
	createCmd.Flags().StringVarP(&p.Version, "version", "v", "", "Version: 0.0.1-SNAPSHOT")
	createCmd.Flags().StringVarP(&p.PackageName, "packagename", "p", "", "Package Name: com.example.demo")
	createCmd.Flags().StringVarP(&p.SpringBootVersion, "springbootversion", "s", "", "Spring Boot Version")
	createCmd.Flags().StringVarP(&p.SnowdropBomVersion, "snowdropbom", "b", "", "Snowdrop Bom Version")

	err := createCmd.Execute()
	if err != nil {
		fmt.Print(err.Error())
	}
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

func getCompatibleModuleNameFor(p *scaffold.Project) []string {
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
