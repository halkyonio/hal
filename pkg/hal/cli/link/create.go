package link

import (
	"fmt"
	"github.com/spf13/cobra"
	"halkyon.io/api/component/v1beta1"
	link "halkyon.io/api/link/v1beta1"
	halkyon "halkyon.io/api/v1beta1"
	"halkyon.io/hal/pkg/cmdutil"
	"halkyon.io/hal/pkg/k8s"
	"halkyon.io/hal/pkg/log"
	"halkyon.io/hal/pkg/ui"
	k8score "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"strings"
	"time"
)

const (
	createCommandName = "create"
	targetSeparator   = ": "
)

type createOptions struct {
	targetName string
	secret     string
	name       string
	envPairs   []string
	envs       []halkyon.NameValuePair
	linkType   link.LinkType
}

func (o *createOptions) Complete(name string, cmd *cobra.Command, args []string) error {
	// first check if proper parameters combination are provided
	useSecret := len(o.secret) > 0
	useEnv := len(o.envPairs) > 0
	if useSecret && useEnv {
		return fmt.Errorf("invalid parameter combination: either pass a secret name or environment variables, not both")
	}

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

	if !useSecret && !useEnv {
		useSecret = ui.Proceed("Use Secret")
	}

	if useSecret {
		o.linkType = link.SecretLinkType
		ui.OutputSelection("Selected link type", o.linkType.String())
		secrets, valid, err := o.checkAndGetValidSecrets()
		if err != nil {
			return err
		}
		if len(secrets) == 0 {
			return fmt.Errorf("no valid secrets currently exist on the cluster")
		}
		if !valid {
			o.secret = ui.Select("Secret (only potential matches shown)", secrets)
		}
	} else {
		o.linkType = link.EnvLinkType
		ui.OutputSelection("Selected link type", o.linkType.String())
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

	generated := fmt.Sprintf("%s-link-%d", o.targetName, time.Now().UnixNano())
	o.name = ui.Ask("Change default name", o.name, generated)

	return nil
}

func (o *createOptions) addToEnv(pair string) (halkyon.NameValuePair, error) {
	// todo: extract as generic version
	split := strings.Split(pair, "=")
	if len(split) != 2 {
		return halkyon.NameValuePair{}, fmt.Errorf("invalid environment variable: %s, format must be 'name=value'", pair)
	}
	env := halkyon.NameValuePair{Name: split[0], Value: split[1]}
	o.envs = append(o.envs, env)
	ui.OutputSelection("Set env variable", fmt.Sprintf("%s=%s", env.Name, env.Value))
	return env, nil
}

func (o *createOptions) Validate() error {
	// todo: validate selected link name
	return nil
}

func (o *createOptions) Run() error {
	client := k8s.GetClient()
	l, err := client.HalkyonLinkClient.Links(client.Namespace).Create(&link.Link{
		ObjectMeta: v1.ObjectMeta{
			Name:      o.name,
			Namespace: client.Namespace,
		},
		Spec: link.LinkSpec{
			ComponentName: o.targetName,
			Type:          o.linkType,
			Ref:           o.secret,
			Envs:          o.envs,
		},
	})

	if err != nil {
		return err
	}

	components := client.HalkyonComponentClient.Components(client.Namespace)
	cp, err := components.Get(o.targetName, v1.GetOptions{})
	if err != nil {
		return err
	}

	// while the pod name hasn't changed, wait
	initialPodName := cp.Status.PodName
	for {
		pod, err := fetchPod(cp)
		if err != nil {
			return err
		}
		if initialPodName == pod.Name {
			time.Sleep(2 * time.Second)
		} else {
			cp.Status.PodName = pod.Name
			break
		}
	}

	cp, err = components.UpdateStatus(cp)
	if err != nil {
		return err
	}

	cp, err = client.WaitForComponent(o.targetName, v1beta1.ComponentReady, "Waiting for link to be ready…")
	if err != nil {
		return fmt.Errorf("error waiting for component: %v", err)
	}
	cp.Status.Message = fmt.Sprintf("'%s' link successfully created", o.name)
	_, err = components.UpdateStatus(cp)
	if err != nil {
		return err
	}

	log.Successf("Created link %s", l.Name)
	// todo:
	//  - read existing application.yml using viper
	//  - merge existing and new link
	//  - write updated application.yml
	return nil
}

func NewCmdCreate(parent string) *cobra.Command {
	o := &createOptions{}
	l := &cobra.Command{
		Use:     fmt.Sprintf("%s [flags]", createCommandName),
		Short:   "Link the current (or target) component to the specified capability or component",
		Long:    `Link the current (or target) component to the specified capability or component`,
		Args:    cobra.NoArgs,
		Example: fmt.Sprintf("  # links the client-sb to the backend-sb component\n %s -n client-to-backend -t client-sb", cmdutil.CommandName(createCommandName, parent)),
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.GenericRun(o, cmd, args)
		},
	}
	l.Flags().StringVarP(&o.targetName, "target", "t", "", "Name of the component or capability to link to")
	l.Flags().StringVarP(&o.name, "name", "n", "", "Link name")
	l.Flags().StringVarP(&o.secret, "secret", "s", "", "Secret name to reference if using Secret type")
	l.Flags().StringSliceVarP(&o.envPairs, "env", "e", []string{}, "Environment variables as 'name=value' pairs")

	return l
}

func (o *createOptions) checkAndGetValidTargets() ([]string, bool, error) {
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

func (createOptions) extractTargetName(typeAndTarget string) string {
	index := strings.Index(typeAndTarget, targetSeparator)
	return typeAndTarget[index+len(targetSeparator):]
}

func (o *createOptions) checkAndGetValidSecrets() ([]string, bool, error) {
	client := k8s.GetClient()
	secrets, err := client.KubeClient.CoreV1().Secrets(client.Namespace).List(v1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("type", string(k8score.SecretTypeOpaque)).String(),
	})
	if err != nil {
		return nil, false, err
	}
	known := make([]string, 0, len(secrets.Items))
	givenIsValid := false
	for _, secret := range secrets.Items {
		known = append(known, secret.Name)
		if secret.Name == o.secret {
			givenIsValid = true
		}
	}
	return known, givenIsValid, nil
}

func fetchPod(instance *v1beta1.Component) (*k8score.Pod, error) {
	client := k8s.GetClient()
	pods, err := client.KubeClient.CoreV1().Pods(instance.Namespace).List(v1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{"app": instance.Name}).String(),
	})
	if err != nil {
		return nil, err
	} else {
		// We assume that there is only one Pod containing the label app=component name AND we return it
		if len(pods.Items) > 0 {
			return &pods.Items[0], nil
		} else {
			err := fmt.Errorf("failed to get pod created for the component")
			return nil, err
		}
	}
}
