package link

import (
	"fmt"
	"github.com/snowdrop/component-api/pkg/apis/component/v1alpha2"
	"github.com/snowdrop/kreate/pkg/cmdutil"
	"github.com/snowdrop/kreate/pkg/k8s"
	"github.com/snowdrop/kreate/pkg/log"
	"github.com/snowdrop/kreate/pkg/ui"
	"github.com/snowdrop/kreate/pkg/validation"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
	"strings"
	"time"
)

const commandName = "link"

type options struct {
	targetName string
	ref        string
	kind       validation.EnumValue
	envPairs   []string
	envs       []v1alpha2.Env
	*cmdutil.ComponentTargetingOptions
}

func (o *options) Complete(name string, cmd *cobra.Command, args []string) error {
	// retrieve and build list of available targets
	capabilitiesAndComponents, validTarget, err := o.checkAndGetValidTargets()
	if err != nil {
		return err
	}
	if len(capabilitiesAndComponents) == 0 {
		return fmt.Errorf("no valid capabilities or components currently exist on the cluster")
	}
	if !validTarget {
		o.targetName = ui.Select("Target", capabilitiesAndComponents)
	}

	if !o.kind.IsProvidedValid() {
		if ui.Proceed("Use Secret") {
			o.kind.MustSet(v1alpha2.SecretLinkKind)
			secrets, valid, err := o.checkAndGetValidSecrets()
			if err != nil {
				return err
			}
			if len(secrets) == 0 {
				return fmt.Errorf("no valid secrets currently exist on the cluster")
			}
			if !valid {
				o.ref = ui.Select("Secret", secrets)
			}
		} else {
			o.kind.MustSet(v1alpha2.EnvLinkKind)
			for _, pair := range o.envPairs {
				if _, e := o.addToEnv(pair); e != nil {
					return e
				}
			}
			for {
				envAsString := ui.AskOrReturnToExit("Env variable in the 'name=value' format, press enter when done")
				if len(envAsString) == 0 {
					break
				}
				if _, e := o.addToEnv(envAsString); e != nil {
					return e
				}
			}
		}
	}
	return nil
}

func (o *options) addToEnv(pair string) (v1alpha2.Env, error) {
	split := strings.Split(pair, "=")
	if len(split) != 2 {
		return v1alpha2.Env{}, fmt.Errorf("invalid environment variable: %s, format must be 'name=value'", pair)
	}
	env := v1alpha2.Env{Name: split[0], Value: split[1]}
	o.envs = append(o.envs, env)
	ui.OutputSelection("Set env variable", fmt.Sprintf("%s=%s", env.Name, env.Value))
	return env, nil
}

func (o *options) Validate() error {
	return o.kind.Contains(o.kind)
}

func (o *options) Run() error {
	name := fmt.Sprintf("%s-link-%d", o.ComponentName, time.Now().UnixNano())
	client := k8s.GetClient()
	link, err := client.DevexpClient.Links(client.Namespace).Create(&v1alpha2.Link{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: client.Namespace,
		},
		Spec: v1alpha2.LinkSpec{
			ComponentName: o.targetName,
			Kind:          o.kind.Get().(v1alpha2.LinkKind),
			Ref:           o.ref,
			Envs:          o.envs,
		},
	})

	if err != nil {
		return err
	}

	log.Successf("Created link %s", link.Name)
	// todo:
	//  - read existing application.yml using viper
	//  - merge existing and new link
	//  - write updated application.yml
	return nil
}

func (o *options) SetTargetingOptions(options *cmdutil.ComponentTargetingOptions) {
	o.ComponentTargetingOptions = options
}

func NewCmdLink(parent string) *cobra.Command {
	o := &options{
		kind: validation.NewEnumValue("kind", v1alpha2.EnvLinkKind, v1alpha2.SecretLinkKind),
	}
	link := &cobra.Command{
		Use:   fmt.Sprintf("%s [flags]", commandName),
		Short: "Link the current (or target) component to the specified capability or component",
		Long:  `Link the current (or target) component to the specified capability or component`,
		Args:  cobra.NoArgs,
	}
	link.Flags().StringVarP(&o.targetName, "target", "t", "", "Name of the component or capability to link to")
	link.Flags().StringVarP(&o.kind.Provided, "kind", "k", "", "Kind of link. Possible values: "+o.kind.GetKnownValues())
	link.Flags().StringSliceVarP(&o.envPairs, "env", "e", []string{}, "Additional environment variables as 'name=value' pairs")

	cmdutil.ConfigureRunnableAndCommandWithTargeting(o, link)
	return link
}

func (o *options) readCurrent() (*v1alpha2.LinkSpec, error) {
	viper.SetConfigName("application")                                              // name of config file (without extension)
	viper.AddConfigPath(filepath.Join(o.ComponentPath, "src", "main", "resources")) // path to look for the config file in
	err := viper.ReadInConfig()                                                     // Find and read the config file
	if err != nil {                                                                 // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
	present := viper.Get("dekorate.link")
	if present != nil {
		link := &v1alpha2.LinkSpec{}
		err = viper.UnmarshalKey("dekorate.link", link)
		if err != nil {
			return nil, err
		}
		return link, nil
	}

	//viper.WriteConfig()
	return nil, nil
}

func (o *options) checkAndGetValidTargets() ([]string, bool, error) {
	const capabilityPrefix = "capability"
	const componentPrefix = "component"
	known := make([]string, 0, 10)
	givenIsValid := false

	client := k8s.GetClient()
	capabilities, err := client.DevexpClient.Capabilities(client.Namespace).List(v1.ListOptions{})
	if err != nil {
		return nil, false, err
	}
	for _, c := range capabilities.Items {
		known = append(known, fmt.Sprintf("%s: %s", capabilityPrefix, c.Name))
		if !givenIsValid && c.Name == o.targetName {
			givenIsValid = true
		}
	}

	components, err := client.DevexpClient.Components(client.Namespace).List(v1.ListOptions{})
	if err != nil {
		return nil, false, err
	}
	for _, c := range components.Items {
		known = append(known, fmt.Sprintf("%s: %s", componentPrefix, c.Name))
		if !givenIsValid && c.Name == o.targetName {
			givenIsValid = true
		}
	}

	return known, givenIsValid, nil
}

func (o *options) checkAndGetValidSecrets() ([]string, bool, error) {
	client := k8s.GetClient()
	secrets, err := client.KubeClient.CoreV1().Secrets(client.Namespace).List(v1.ListOptions{})
	if err != nil {
		return nil, false, err
	}
	known := make([]string, 0, len(secrets.Items))
	givenIsValid := false
	for _, secret := range secrets.Items {
		known = append(known, secret.Name)
		if secret.Name == o.ref {
			givenIsValid = true
		}
	}
	return known, givenIsValid, nil
}
