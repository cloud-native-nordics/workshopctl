package cmd

import (
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// NewKubectlCommand returns the "kubectl" command
func NewKubectlCommand(in io.Reader, out, err io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kubectl",
		Short: "An alias for the kubectl command, pointing the KUBECONFIG to the right place",
		RunE:  RunKubectl,
	}

	addKubectlFlags(cmd.Flags())
	return cmd
}

func addKubectlFlags(fs *pflag.FlagSet) {

}

func RunKubectl(cmd *cobra.Command, args []string) error {
	// Add stuff here
	return nil
}
