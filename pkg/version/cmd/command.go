package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/cloud-native-nordics/workshopctl/pkg/version"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

// NewCmdVersion provides the version information of ignite
func NewCmdVersion(out io.Writer) *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunVersion(out, output)
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", output, "Output format; available options are 'yaml', 'json' and 'short'")
	return cmd
}

// RunVersion provides the version information for the specified format
func RunVersion(out io.Writer, output string) error {
	v := version.Get()
	switch output {
	case "":
		fmt.Fprintf(out, "Version: %#v\n", v)
	case "short":
		fmt.Fprintf(out, "%s\n", v)
	case "yaml":
		y, err := yaml.Marshal(&v)
		if err != nil {
			return err
		}
		fmt.Fprintln(out, string(y))
	case "json":
		y, err := json.MarshalIndent(&v, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(out, string(y))
	default:
		return fmt.Errorf("invalid output format: %s", output)
	}

	return nil
}
