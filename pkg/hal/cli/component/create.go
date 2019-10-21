package component

import (
	"fmt"
	"github.com/spf13/cobra"
	"halkyon.io/api/component/v1beta1"
	"halkyon.io/hal/pkg/cmdutil"
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
	"text/template"
)

// todo: remove and replace by operator querying
var runtimes = map[string]halkyonRuntime{
	"spring-boot": {
		name:      "spring-boot",
		versions:  []string{"2.1.6.RELEASE", "1.5.19.RELEASE"},
		generator: `https://generator.snowdrop.me/app?springbootversion={{.RV}}&groupid={{.G}}&artifactid={{.A}}&version={{.V}}&template={{.Template}}&packagename={{.P}}&outdir={{.Name}}`,
	},
	"quarkus": {
		name:      "quarkus",
		versions:  []string{"0.23.2"},
		generator: `https://code.quarkus.io/api/download?g={{.G}}&a={{.A}}&v={{.V}}&c={{.P}}.ResourceExample`,
	},
	"vert.x": {
		name:      "vert.x",
		versions:  []string{"3.8.2", "3.7.1"},
		generator: `https://start.vertx.io/starter.zip?vertxVersion={{.RV}}&groupId={{.G}}&artifactId={{.A}}&packageName={{.P}}`,
	},
	"thorntail": {name: "thorntail", versions: []string{"2.5.0.Final", "2.4.0.Final"}},
	"node.js":   {name: "node.js", versions: []string{"12.x", "10.x", "8.x"}},
}

type halkyonRuntime struct {
	name      string
	versions  []string
	generator string
}

type createOptions struct {
	*cmdutil.CreateOptions
	*cmdutil.EnvOptions
	runtime   string
	RV        string
	exposeP   string
	expose    bool
	port      int
	scaffold  bool
	G         string
	A         string
	V         string
	generator string
	Template  string
	P         string
	scaffoldP string
}

func (o *createOptions) GeneratePrefix() string {
	return o.runtime
}

func (o *createOptions) Build() runtime.Object {
	if len(o.generator) > 0 {
		err := io.Generate(o.generator, o.Name)
		if err != nil {
			panic(err)
		}
	}
	return &v1beta1.Component{
		ObjectMeta: v1.ObjectMeta{
			Name:      o.Name,
			Namespace: o.CreateOptions.Client.GetNamespace(),
		},
		Spec: v1beta1.ComponentSpec{
			Runtime:       o.runtime,
			Version:       o.RV,
			ExposeService: o.expose,
			Port:          int32(o.port),
			Envs:          o.Envs,
		},
	}
}

var (
	createExample = ktemplates.Examples(`  # Create a new Halkyon component located in the 'foo' child directory of the current directory
  %[1]s foo`)
)

func (o *createOptions) Complete(name string, cmd *cobra.Command, args []string) error {
	ui.SelectOrCheckExisting(&o.runtime, "Runtime", o.getRuntimes(), o.isValidRuntime)
	ui.SelectOrCheckExisting(&o.RV, "Version", o.getVersionsForRuntime(), o.isValidVersionGivenRuntime)

	if len(o.exposeP) == 0 {
		o.expose = ui.Proceed("Expose microservice")
	} else {
		b, err := strconv.ParseBool(o.exposeP)
		if err != nil {
			return err
		}
		o.expose = b
	}

	if o.expose && o.port == 0 {
		port := ui.Ask("Port", fmt.Sprintf("%d", o.port), "8080")
		intPort, err := strconv.Atoi(port)
		if err != nil {
			return err
		}
		o.port = intPort
	}

	if len(o.scaffoldP) == 0 {
		o.scaffold = ui.Proceed("Use code generator")
	} else {
		b, err := strconv.ParseBool(o.scaffoldP)
		if err != nil {
			return err
		}
		o.scaffold = b
	}

	r := runtimes[o.runtime]
	if len(r.generator) > 0 && o.scaffold {
		o.G = ui.Ask("Group Id", o.G, "dev.snowdrop")
		o.A = ui.Ask("Artifact Id", o.A, "myproject")
		o.V = ui.Ask("Version", o.V, "1.0.0-SNAPSHOT")
		o.P = ui.Ask("Package name", o.P, o.G+"."+o.A)
		o.generator = r.generator // set the generator url to the unparsed runtime generator url to be filled in Validate
		o.scaffold = true
	} else {
		o.scaffold = false
		names := o.getChildDirNames()
		if len(names) > 0 {
			ui.SelectOrCheckExisting(&o.Name, "Local component directory", names, func() bool { return true })
		}
	}

	if err := o.EnvOptions.Complete(name, cmd, args); err != nil {
		return err
	}

	return nil
}

func (o *createOptions) Validate() error {
	matched, err := regexp.MatchString("^([a-zA-Z][a-zA-Z\\d_]*\\.)*", o.P)
	if !matched {
		msg := ""
		if err != nil {
			msg = ": " + err.Error()
		}
		return fmt.Errorf("'%s' is an invalid package name%s", o.P, msg)
	}
	currentDir, _ := os.Getwd()
	children := o.getChildDirNames()
	if o.scaffold {
		// generate the generator URL since we need to make sure that all fields are set (in particular Name) before executing
		// complete generator URL:
		t := template.New("generator")
		parsed, err := t.Parse(o.generator)
		if err != nil {
			return err
		}
		builder := &strings.Builder{}
		err = parsed.Execute(builder, o)
		if err != nil {
			return err
		}
		o.generator = builder.String()

		// a directory will be created by the scaffolding process, we need to check that it won't override an existing dir
		for _, child := range children {
			if o.Name == child {
				return fmt.Errorf("a directory named '%s' already exists in %s", o.Name, currentDir)
			}
		}
		return nil
	} else if !validation.IsValidDir(o.Name) {
		if len(children) == 0 {
			// if we're not scaffolding and we don't have any existing children directory, create one
			err := os.Mkdir(o.Name, os.ModePerm)
			if err != nil {
				return err
			}
			ui.OutputSelection("Created new component directory", o.Name)
		} else {
			return fmt.Errorf("no directory named '%s' exists in %v", o.Name, currentDir)
		}
	}
	return nil
}

func (o *createOptions) getRuntimes() []string {
	// todo: implement operator querying
	return getRuntimeNames()
}

func (o *createOptions) isValidRuntime() bool {
	return validation.IsValid(o.runtime, o.getRuntimes())
}

func (o *createOptions) getVersionsForRuntime() []string {
	// todo: implement operator querying
	r, ok := runtimes[o.runtime]
	if !ok {
		return []string{"Unknown runtime " + o.runtime} // shouldn't happen
	}
	return r.versions
}

func (o *createOptions) isValidVersionGivenRuntime() bool {
	return validation.IsValid(o.RV, o.getVersionsForRuntime())
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
	c := k8s.GetClient()
	o := &createOptions{}
	generic := cmdutil.NewCreateOptions(cmdutil.Component, client{
		client: c.HalkyonComponentClient.Components(c.Namespace),
		ns:     c.Namespace,
	})
	generic.Delegate = o
	o.CreateOptions = generic
	cmd := cmdutil.NewGenericCreate(fullParentName, generic)
	cmd.Example = fmt.Sprintf(createExample, cmdutil.CommandName(cmd.Name(), fullParentName))

	cmd.Flags().StringVarP(&o.runtime, "runtime", "r", "", "Runtime to use for the component. Possible values:"+strings.Join(getRuntimeNames(), ","))
	cmd.Flags().StringVarP(&o.RV, "runtimeVersion", "i", "", "Runtime version")
	cmd.Flags().StringVarP(&o.exposeP, "expose", "x", "", "Whether or not to expose the microservice outside of the cluster")
	cmd.Flags().IntVarP(&o.port, "port", "o", 0, "Port the microservice listens on")
	cmd.Flags().StringVarP(&o.scaffoldP, "scaffold", "s", "", "Use code generator to scaffold the component")
	cmd.Flags().StringVarP(&o.G, "groupid", "g", "", "Maven group id e.g. com.example")
	cmd.Flags().StringVarP(&o.A, "artifactid", "a", "", "Maven artifact id e.g. demo")
	cmd.Flags().StringVarP(&o.V, "version", "v", "", "Maven version e.g. 0.0.1-SNAPSHOT")
	cmd.Flags().StringVarP(&o.Template, "template", "t", "rest", "Template name used to select the project to be created, only supported for Spring Boot")
	cmd.Flags().StringVarP(&o.P, "packagename", "p", "", "Package name (defaults to <group id>.<artifact id>)")

	cmdutil.SetupEnvOptions(o, cmd)

	return cmd
}
