package cmd

import (
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// NewInitCommand returns the "init" command
func NewInitCommand(in io.Reader, out, err io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Setup the user configuration interactively",
		RunE:  RunInit,
	}

	addInitFlags(cmd.Flags())
	return cmd
}

func addInitFlags(fs *pflag.FlagSet) {

}

func RunInit(cmd *cobra.Command, args []string) error {
	// Add stuff here
	return nil
}
