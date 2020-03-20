package component

import (
	"fmt"
	"github.com/spf13/cobra"
	v1beta13 "halkyon.io/api/capability/v1beta1"
	"halkyon.io/api/component/v1beta1"
	v1beta12 "halkyon.io/api/runtime/v1beta1"
	halkyon "halkyon.io/api/v1beta1"
	"halkyon.io/hal/pkg/cmdutil"
	"halkyon.io/hal/pkg/hal/cli/capability"
	"halkyon.io/hal/pkg/io"
	"halkyon.io/hal/pkg/k8s"
	"halkyon.io/hal/pkg/ui"
	"halkyon.io/hal/pkg/validation"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ktemplates "k8s.io/kubectl/pkg/util/templates"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var runtimes = <-getRuntimes()

type halkyonRuntime struct {
	name      string
	versions  []string
	generator string
}

type createOptions struct {
	*cmdutil.CreateOptions
	*cmdutil.EnvOptions
	*v1beta12.GeneratorOptions
	runtime      string
	exposeP      string
	expose       bool
	port         int
	scaffold     bool
	generator    string
	scaffoldP    string
	requiredCaps []v1beta1.RequiredCapabilityConfig
	providedCaps []v1beta1.CapabilityConfig
	target       *v1beta1.Component
}

func (o *createOptions) GeneratePrefix() string {
	return o.runtime
}

func (o *createOptions) Build() runtime.Object {
	if o.target == nil {
		if len(o.generator) > 0 {
			err := io.Generate(o.generator, o.Name)
			if err != nil {
				panic(err)
			}
		}

		o.target = &v1beta1.Component{
			ObjectMeta: v1.ObjectMeta{
				Name:      o.Name,
				Namespace: o.CreateOptions.Client.GetNamespace(),
			},
			Spec: v1beta1.ComponentSpec{
				Runtime:       o.runtime,
				Version:       o.RuntimeVersion,
				ExposeService: o.expose,
				Port:          int32(o.port),
				Capabilities: v1beta1.CapabilitiesConfig{
					Requires: o.requiredCaps,
					Provides: o.providedCaps,
				},
				Envs: o.Envs,
			},
		}
	}

	return o.target
}

func (o *createOptions) Set(entity runtime.Object) {
	o.target = entity.(*v1beta1.Component)
}

var (
	createExample = ktemplates.Examples(`  # Create a new Halkyon component located in the 'foo' child directory of the current directory
  %[1]s foo`)
)

func (o *createOptions) Complete(name string, cmd *cobra.Command, args []string) error {
	ui.SelectOrCheckExisting(&o.runtime, "Runtime", o.getRuntimes(), o.isValidRuntime)
	ui.SelectOrCheckExisting(&o.RuntimeVersion, "Version", o.getVersionsForRuntime(), o.isValidVersionGivenRuntime)

	if len(o.exposeP) == 0 {
		o.expose = ui.Proceed("Expose microservice")
	} else {
		b, err := strconv.ParseBool(o.exposeP)
		if err != nil {
			return err
		}
		o.expose = b
	}

	if o.port == 0 {
		port := ui.Ask("Port", fmt.Sprintf("%d", o.port), "8080")
		intPort, err := strconv.Atoi(port)
		if err != nil {
			return err
		}
		o.port = intPort
	}

	r := runtimes[o.runtime]
	hasGenerator := len(r.generator) > 0
	if len(o.scaffoldP) == 0 {
		o.scaffold = hasGenerator && ui.Proceed("Use code generator")
	} else {
		b, err := strconv.ParseBool(o.scaffoldP)
		if err != nil {
			return err
		}
		if b && !hasGenerator {
			ui.OutputError(fmt.Sprintf("ignoring scaffolding option because unsupported by %s runtime", r.name))
		}
		o.scaffold = hasGenerator && b
	}

	if o.scaffold {
		o.GroupId = ui.Ask("Group Id", o.GroupId, "dev.snowdrop")
		o.ArtifactId = ui.Ask("Artifact Id", o.ArtifactId, "myproject")
		o.ProjectVersion = ui.Ask("Version", o.ProjectVersion, "1.0.0-SNAPSHOT")
		o.PackageName = ui.Ask("Package name", o.PackageName, o.GroupId+"."+o.ArtifactId)
		o.generator = r.generator // set the generator url to the unparsed runtime generator url to be filled in Validate
		o.scaffold = true
	} else {
		o.scaffold = false
		names := o.getChildDirNames()
		if len(names) > 0 {
			ui.SelectOrCheckExisting(&o.Name, "Local component directory", names, func() bool { return true })
		}
	}

	if ui.Proceed("Requires capabilities") {
		required := v1beta1.RequiredCapabilityConfig{}
		o.requiredCaps = make([]v1beta1.RequiredCapabilityConfig, 0, 10)
		existing := capability.Entity.GetMatching()
		hasCaps := len(existing) > 0
		for {
			required.Name = ui.AskOrReturnToExit("Required capability name, simply press enter to finish")
			if len(required.Name) == 0 {
				break
			}
			if hasCaps && ui.Proceed("Bind to existing capability") {
				required.BoundTo = ui.Select("Target capability", getCapabilityNames(existing))
				required.Spec = existing[required.BoundTo]
			} else {
				capCreate := capability.CapabilityCreateOptions{}
				if err := capCreate.Complete(); err != nil {
					return err
				}
				required.Spec = capCreate.AsCapabilitySpec()
				required.AutoBindable = ui.Proceed("Auto-bindable")
			}
			if ui.Proceed("Add extra parameters") {
				for {
					paramPair := ui.AskOrReturnToExit("Parameter in the 'name=value' format, simply press enter when finished")
					if len(paramPair) == 0 {
						break
					}
					split := strings.Split(paramPair, "=")
					if len(split) != 2 {
						return fmt.Errorf("invalid parameter: %s, format must be 'name=value'", paramPair)
					}
					param := halkyon.NameValuePair{Name: split[0], Value: split[1]}
					required.Spec.Parameters = append(required.Spec.Parameters, param)
					ui.OutputSelection("Set parameter", fmt.Sprintf("%s=%s", param.Name, param.Value))
				}
			}
			o.requiredCaps = append(o.requiredCaps, required)
		}
	}

	if ui.Proceed("Provides capabilities") {
		provided := v1beta1.CapabilityConfig{}
		o.providedCaps = make([]v1beta1.CapabilityConfig, 0, 10)
		for {
			provided.Name = ui.AskOrReturnToExit("Provided capability name, simply press enter to finish")
			if len(provided.Name) == 0 {
				break
			}
			capCreate := capability.CapabilityCreateOptions{}
			if err := capCreate.Complete(); err != nil {
				return err
			}
			if err := capCreate.Validate(); err != nil {
				return err
			}
			provided.Spec = capCreate.AsCapabilitySpec()
			o.providedCaps = append(o.providedCaps, provided)
		}
	}

	if err := o.EnvOptions.Complete(name, cmd, args); err != nil {
		return err
	}

	return nil
}

func getCapabilityNames(caps map[string]v1beta13.CapabilitySpec) []string {
	result := make([]string, 0, len(caps))
	for k := range caps {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}

func (o *createOptions) Validate() error {
	matched, err := regexp.MatchString("^([a-zA-Z][a-zA-Z\\d_]*\\.)*", o.PackageName)
	if !matched {
		msg := ""
		if err != nil {
			msg = ": " + err.Error()
		}
		return fmt.Errorf("'%s' is an invalid package name%s", o.PackageName, msg)
	}
	currentDir, _ := os.Getwd()
	children := o.getChildDirNames()
	if o.scaffold {
		// generate the generator URL since we need to make sure that all fields are set (in particular Name) before executing
		// complete generator URL:
		o.generator, err = v1beta12.ComputeGeneratorURL(o.generator, *o.GeneratorOptions)
		if err != nil {
			return err
		}

		// a directory will be created by the scaffolding process, we need to check that it won't override an existing dir
		for _, child := range children {
			if o.Name == child {
				return fmt.Errorf("a directory named '%s' already exists in %s", o.Name, currentDir)
			}
		}
		return nil
	} else if !validation.IsValidDir(o.Name) {
		if len(children) == 0 || ui.Proceed(fmt.Sprintf("no directory named '%s' exists in %v, create it", o.Name, currentDir)) {
			// if we're not scaffolding and we don't have any existing children directory, create one
			err := os.Mkdir(o.Name, os.ModePerm)
			if err != nil {
				return err
			}
			ui.OutputSelection("Created new component directory", o.Name)
		} else {
			return fmt.Errorf("'%s' directory was not created in %v", o.Name, currentDir)
		}
	}
	return nil
}

func (o *createOptions) getRuntimes() []string {
	return getRuntimeNames()
}

func (o *createOptions) isValidRuntime() bool {
	return validation.IsValid(o.runtime, o.getRuntimes())
}

func (o *createOptions) getVersionsForRuntime() []string {
	r, ok := runtimes[o.runtime]
	if !ok {
		return []string{"Unknown runtime " + o.runtime} // shouldn't happen
	}
	return r.versions
}

func (o *createOptions) isValidVersionGivenRuntime() bool {
	return validation.IsValid(o.RuntimeVersion, o.getVersionsForRuntime())
}

func (o *createOptions) getChildDirNames() []string {
	currentDir, _ := os.Getwd()
	childDirs := make([]string, 0, 7)
	children, err := ioutil.ReadDir(currentDir)
	if err != nil {
		panic(err)
	}
	for _, child := range children {
		if child.IsDir() {
			childDirs = append(childDirs, child.Name())
		}
	}
	return childDirs
}

func getRuntimes() chan map[string]*halkyonRuntime {
	r := make(chan map[string]*halkyonRuntime)

	go func() {
		list, err := k8s.GetClient().HalkyonRuntimeClient.Runtimes().List(v1.ListOptions{})
		if err != nil {
			panic(err)
		}

		hRuntimes := make(map[string]*halkyonRuntime, 11)
		for _, item := range list.Items {
			name := item.Spec.Name
			runtime, ok := hRuntimes[name]
			if !ok {
				runtime = &halkyonRuntime{name: name, generator: item.Spec.GeneratorTemplate}
				hRuntimes[name] = runtime
			}

			versions := runtime.versions
			if len(versions) == 0 {
				versions = make([]string, 0, 7)
			}
			versions = append(versions, item.Spec.Version)
			runtime.versions = versions
		}

		r <- hRuntimes
	}()

	return r
}

func getRuntimeNames() []string {
	result := make([]string, 0, len(runtimes))
	for k := range runtimes {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}

func (o *createOptions) SetEnvOptions(env *cmdutil.EnvOptions) {
	o.EnvOptions = env
}

func NewCmdCreate(fullParentName string) *cobra.Command {
	o := &createOptions{}
	generic := cmdutil.NewCreateOptions(cmdutil.Component, Entity)
	generic.Delegate = o
	o.CreateOptions = generic
	o.GeneratorOptions = &v1beta12.GeneratorOptions{}
	cmd := cmdutil.NewGenericCreate(fullParentName, generic)
	cmd.Example = fmt.Sprintf(createExample, cmdutil.CommandName(cmd.Name(), fullParentName))

	cmd.Flags().StringVarP(&o.runtime, "runtime", "r", "", "Runtime to use for the component. Possible values:"+strings.Join(getRuntimeNames(), ","))
	cmd.Flags().StringVarP(&o.RuntimeVersion, "runtimeVersion", "i", "", "Runtime version")
	cmd.Flags().StringVarP(&o.exposeP, "expose", "x", "", "Whether or not to expose the microservice outside of the cluster")
	cmd.Flags().IntVarP(&o.port, "port", "o", 0, "Port the microservice listens on")
	cmd.Flags().StringVarP(&o.scaffoldP, "scaffold", "s", "", "Use code generator to scaffold the component")
	cmd.Flags().StringVarP(&o.GroupId, "groupid", "g", "", "Maven group id e.g. com.example")
	cmd.Flags().StringVarP(&o.ArtifactId, "artifactid", "a", "", "Maven artifact id e.g. demo")
	cmd.Flags().StringVarP(&o.ProjectVersion, "version", "v", "", "Maven version e.g. 0.0.1-SNAPSHOT")
	cmd.Flags().StringVarP(&o.ProjectTemplate, "template", "t", "rest", "Template name used to select the project to be created, only supported for Spring Boot")
	cmd.Flags().StringVarP(&o.PackageName, "packagename", "p", "", "Package name (defaults to <group id>.<artifact id>)")

	cmdutil.SetupEnvOptions(o, cmd)

	return cmd
}
