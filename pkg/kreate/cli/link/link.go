package link

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	link "halkyon.io/api/link/v1beta1"
	halkyon "halkyon.io/api/v1beta1"
	"halkyon.io/kreate/pkg/cmdutil"
	"halkyon.io/kreate/pkg/k8s"
	"halkyon.io/kreate/pkg/log"
	"halkyon.io/kreate/pkg/ui"
	"halkyon.io/kreate/pkg/validation"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
	"strings"
	"time"
)

const (
	commandName     = "link"
	targetSeparator = ": "
)

type options struct {
	targetName string
	ref        string
	kind       validation.EnumValue
	envPairs   []string
	envs       []halkyon.Env
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
		o.targetName = o.extractTargetName(ui.Select("Target", capabilitiesAndComponents))
	}

	if !o.kind.IsProvidedValid() {
		if ui.Proceed("Use Secret") {
			o.kind.MustSet(link.SecretLinkType)
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
			o.kind.MustSet(link.EnvLinkType)
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

func (o *options) addToEnv(pair string) (halkyon.Env, error) {
	split := strings.Split(pair, "=")
	if len(split) != 2 {
		return halkyon.Env{}, fmt.Errorf("invalid environment variable: %s, format must be 'name=value'", pair)
	}
	env := halkyon.Env{Name: split[0], Value: split[1]}
	o.envs = append(o.envs, env)
	ui.OutputSelection("Set env variable", fmt.Sprintf("%s=%s", env.Name, env.Value))
	return env, nil
}

func (o *options) Validate() error {
	return o.kind.Contains(o.kind)
}

func (o *options) Run() error {
	client := k8s.GetClient()
	link, err := client.HalkyonLinkClient.Links(client.Namespace).Create(&link.Link{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: client.Namespace,
		},
		Spec: link.LinkSpec{
			ComponentName: o.targetName,
			Type:          o.kind.Get().(link.LinkType),
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
		kind: validation.NewEnumValue("kind", link.EnvLinkType, link.SecretLinkType),
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

func (o *options) readCurrent() (*link.LinkSpec, error) {
	viper.SetConfigName("application")                                              // name of config file (without extension)
	viper.AddConfigPath(filepath.Join(o.ComponentPath, "src", "main", "resources")) // path to look for the config file in
	err := viper.ReadInConfig()                                                     // Find and read the config file
	if err != nil {                                                                 // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
	present := viper.Get("dekorate.link")
	if present != nil {
		link := &link.LinkSpec{}
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
	capabilities, err := client.HalkyonCapabilityClient.Capabilities(client.Namespace).List(v1.ListOptions{})
	if err != nil {
		return nil, false, err
	}
	for _, c := range capabilities.Items {
		known = append(known, fmt.Sprintf("%s%s%s", capabilityPrefix, targetSeparator, c.Name))
		if !givenIsValid && c.Name == o.targetName {
			givenIsValid = true
		}
	}

	components, err := client.HalkyonComponentClient.Components(client.Namespace).List(v1.ListOptions{})
	if err != nil {
		return nil, false, err
	}
	for _, c := range components.Items {
		known = append(known, fmt.Sprintf("%s%s%s", componentPrefix, targetSeparator, c.Name))
		if !givenIsValid && c.Name == o.targetName {
			givenIsValid = true
		}
	}

	return known, givenIsValid, nil
}

func (options) extractTargetName(typeAndTarget string) string {
	index := strings.Index(typeAndTarget, targetSeparator)
	return typeAndTarget[index+len(targetSeparator):]
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
