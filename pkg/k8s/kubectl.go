package k8s

import (
	"fmt"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubernetes/pkg/kubectl/cmd/apply"
	"k8s.io/kubernetes/pkg/kubectl/cmd/cp"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"os"
)

func Copy(path, namespace, destination string) error {
	c := cp.NewCmdCp(cmdParams())
	c.SetArgs([]string{path, fmt.Sprintf("%s:/deployments/app.jar", destination), "-n", namespace})

	return exec(c)
}

func Apply(path, namespace string) error {
	f, ioStreams := cmdParams()
	c := apply.NewCmdApply("kreate", f, ioStreams)
	c.SetArgs([]string{"apply", "-f", path, "-n", namespace})
	return exec(c)
}

func cmdParams() (f cmdutil.Factory, ioStreams genericclioptions.IOStreams) {
	kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	matchVersionKubeConfigFlags := cmdutil.NewMatchVersionFlags(kubeConfigFlags)
	f = cmdutil.NewFactory(matchVersionKubeConfigFlags)
	ioStreams = genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}

	return f, ioStreams
}

func exec(cmd *cobra.Command) error {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return err
	}
	return nil
}
