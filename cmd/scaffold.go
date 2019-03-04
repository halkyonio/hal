package main

import (
	"archive/zip"
	"fmt"
	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	"github.com/snowdrop/odo-scaffold-plugin/pkg/scaffold"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	ServiceEndpoint = "http://spring-boot-generator.195.201.87.126.nip.io"
)

// HandleError handles UI-related errors, in particular useful to gracefully handle ctrl-c interrupts gracefully
func HandleError(err error) {
	if err != nil {
		if err == terminal.InterruptErr {
			os.Exit(1)
		} else {
			fmt.Printf("Encountered an error processing prompt: %v", err)
		}
	}
}

// Proceed displays a given message and asks the user if they want to proceed
func Proceed(message string) bool {
	var response bool
	prompt := &survey.Confirm{
		Message: message,
	}

	err := survey.AskOne(prompt, &response, survey.Required)
	HandleError(err)

	return response
}

func Select(message string, options []string, defaultValue ...string) string {
	prompt := &survey.Select{
		Message: message,
		Options: options,
	}
	if len(defaultValue) == 1 {
		prompt.Default = defaultValue[0]
	}
	return askOne(prompt)
}

func MultiSelect(message string, options []string) []string {
	modules := []string{}
	prompt := &survey.MultiSelect{
		Message: message,
		Options: options,
	}
	err := survey.AskOne(prompt, &modules, survey.Required)
	HandleError(err)
	return modules
}

func Ask(message string, defaultValue ...string) string {
	input := &survey.Input{
		Message: message,
	}

	if len(defaultValue) == 1 {
		input.Default = defaultValue[0]
	}

	return askOne(input)
}

func askOne(prompt survey.Prompt, stdio ...terminal.Stdio) string {
	var response string

	err := survey.AskOne(prompt, &response, survey.Required)
	HandleError(err)

	return response
}

func main() {
	p := &scaffold.Project{}

	createCmd := &cobra.Command{
		Use:   "scaffold [flags]",
		Short: "Create a Spring Boot maven project",
		Long:  `Create a Spring Boot maven project.`,
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := GetGeneratorServiceConfig(p.UrlService)

			if Proceed("Create from template") {
				p.Template = Select("Available templates", c.GetTemplateNames())
			} else {
				p.Modules = MultiSelect("Select modules", c.GetModuleNames())
			}

			p.GroupId = Ask("Group Id", "me.snowdrop")
			p.ArtifactId = Ask("Artifact Id", "myproject")
			p.Version = Ask("Version", "1.0.0-SNAPSHOT")
			p.PackageName = Ask("Package name", p.GroupId+"."+p.ArtifactId)
			versions, defaultVersion := c.GetSpringBootVersions()
			p.SpringBootVersion = Select("Spring Boot version", versions, defaultVersion)
			if Proceed("Use supported version") {
				p.SnowdropBomVersion = c.GetSupportedVersionFor(p.SpringBootVersion)
			}

			currentDir, _ := os.Getwd()
			p.OutDir = Ask(fmt.Sprintf("Project location (immediate child directory of %s)", currentDir))

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

func GetGeneratorServiceConfig(url string) *scaffold.Config {
	// Call the /config endpoint to get the configuration
	URL := strings.Join([]string{url, "config"}, "/")
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
		log.Fatal("Generator service is not able to find the config !")
	}

	c := &scaffold.Config{}
	err = yaml.Unmarshal(body, &c)
	if err != nil {
		log.Fatal(err.Error())
	}

	return c
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
