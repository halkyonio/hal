package cmdutil

import (
	"fmt"
	"github.com/spf13/cobra"
	halkyon "halkyon.io/api/v1beta1"
	"halkyon.io/hal/pkg/ui"
	"strings"
)

type WithEnv interface {
	SetEnvOptions(o *EnvOptions)
}

type EnvOptions struct {
	EnvPairs []string
	Envs     []halkyon.NameValuePair
}

func SetupEnvOptions(o WithEnv, cmd *cobra.Command) {
	env := &EnvOptions{}
	o.SetEnvOptions(env)
	cmd.Flags().StringSliceVarP(&env.EnvPairs, "env", "e", []string{}, "Environment variables as 'name=value' pairs")
}

func (o *EnvOptions) Complete() error {
	if len(o.EnvPairs) > 0 {
		for _, pair := range o.EnvPairs {
			if _, e := o.addToEnv(pair); e != nil {
				return e
			}
		}
	} else {
		for {
			envAsString := ui.AskOrReturnToExit("Env variable in the 'name=value' format, simply press enter when finished")
			if len(envAsString) == 0 {
				break
			}
			if _, e := o.addToEnv(envAsString); e != nil {
				return e
			}
		}
	}
	return nil
}

func (o *EnvOptions) addToEnv(pair string) (halkyon.NameValuePair, error) {
	// todo: extract as generic version
	split := strings.Split(pair, "=")
	if len(split) != 2 {
		return halkyon.NameValuePair{}, fmt.Errorf("invalid environment variable: %s, format must be 'name=value'", pair)
	}
	env := halkyon.NameValuePair{Name: split[0], Value: split[1]}
	o.Envs = append(o.Envs, env)
	ui.OutputSelection("Set env variable", fmt.Sprintf("%s=%s", env.Name, env.Value))
	return env, nil
}
