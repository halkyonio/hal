package component

import (
	"fmt"
	"github.com/spf13/cobra"
	"halkyon.io/api/component/v1beta1"
	"halkyon.io/hal/pkg/cmdutil"
	"halkyon.io/hal/pkg/k8s"
	"halkyon.io/hal/pkg/ui"
	"halkyon.io/hal/pkg/validation"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ktemplates "k8s.io/kubectl/pkg/util/templates"
	"os"
	"sort"
	"strconv"
	"strings"
)

// todo: remove and replace by operator querying
var runtimes = map[string][]string{
	"spring-boot": {"1.5.19.RELEASE", "2.1.2.RELEASE", "2.1.3.RELEASE", "2.1.6.RELEASE"},
	"vert.x":      {"3.7.1", "3.8.0", "3.8.1"},
	"thorntail":   {"2.4.0.Final", "2.5.0.Final"},
	"node.js":     {"8.x", "10.x", "12.x"},
}

type createOptions struct {
	*cmdutil.CreateOptions
	runtime string
	version string
	expose  bool
	port    int32
}

func (o *createOptions) GeneratePrefix() string {
	return o.runtime
}

func (o *createOptions) Build() runtime.Object {
	return &v1beta1.Component{
		ObjectMeta: v1.ObjectMeta{
			Name:      o.Name,
			Namespace: o.CreateOptions.Client.GetNamespace(),
		},
		Spec: v1beta1.ComponentSpec{
			Runtime:       o.runtime,
			Version:       o.version,
			ExposeService: o.expose,
			Port:          o.port,
		},
	}
}

var (
	createExample = ktemplates.Examples(`  # Create a new Halkyon component found in the 'foo' child directory of the current directory
  %[1]s --name foo`)
)

func (o *createOptions) Complete(name string, cmd *cobra.Command, args []string) error {
	ui.SelectOrCheckExisting(&o.runtime, "Runtime", o.getRuntimes(), o.isValidRuntime)
	ui.SelectOrCheckExisting(&o.version, "Version", o.getVersionsForRuntime(), o.isValidVersionGivenRuntime)
	o.expose = ui.Proceed("Expose microservice")
	port := ui.Ask("Port", fmt.Sprintf("%d", o.port), "8080")
	intPort, err := strconv.Atoi(port)
	if err != nil {
		return err
	}
	o.port = int32(intPort)
	ui.SelectOrCheckExisting(&o.Name, "Local component directory", o.getChildDirNames(), o.isValidComponentName)
	return validation.IntegerValidator(port)
}

func (o *createOptions) Validate() error {
	if !o.isValidComponentName() {
		currentDir, _ := os.Getwd()
		return fmt.Errorf("no directory named '%s' exists in %v", o.Name, currentDir)
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
	versions, ok := runtimes[o.runtime]
	if !ok {
		return []string{"Unknown runtime " + o.runtime} // shouldn't happen
	}
	return versions
}

func (o *createOptions) isValidVersionGivenRuntime() bool {
	return validation.IsValid(o.version, o.getVersionsForRuntime())
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

func (o *createOptions) isValidComponentName() bool {
	return validation.IsValidDir(o.Name) && validation.NameValidator(o.Name) == nil
}

func getRuntimeNames() []string {
	result := make([]string, 0, len(runtimes))
	for k := range runtimes {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}

func NewCmdCreate(fullParentName string) *cobra.Command {
	c := k8s.GetClient()
	o := &createOptions{}
	generic := cmdutil.NewCreateOptions("component", client{
		client: c.HalkyonComponentClient.Components(c.Namespace),
		ns:     c.Namespace,
	})
	generic.Delegate = o
	o.CreateOptions = generic
	cmd := cmdutil.NewGenericCreate(fullParentName, generic)
	cmd.Example = fmt.Sprintf(createExample, cmdutil.CommandName(cmd.Name(), fullParentName))

	cmd.Flags().StringVarP(&o.runtime, "runtime", "r", "", "Runtime to use for the component. Possible values:"+strings.Join(getRuntimeNames(), ","))
	cmd.Flags().StringVarP(&o.version, "version", "v", "", "Runtime version")
	cmd.Flags().BoolVarP(&o.expose, "expose", "e", true, "Whether or not to expose the microservice outside of the cluster, defaults to 'true'")
	cmd.Flags().Int32VarP(&o.port, "port", "p", 0, "Port the microservice listens on, defaults to 8080")
	return cmd
}
