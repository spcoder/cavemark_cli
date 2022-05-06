package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	activateDeployKey string
)

var activateCmd = &cobra.Command{
	Use:   "activate",
	Short: "activate a deployment",
	Long: `Activates an Cavemark deployment.

Example:
  # activates the code running in the 'example' deployment at https://example.com
  cavemark activate -u https://example.com -k example`,
	Args: cobra.MaximumNArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		printActivateHeader(cmd.Parent().Version)
		return activateDeployment(activateDeployKey)
	},
}

func printActivateHeader(version string) {
	p("cavemark", "version %s\n", version)
	p("cavemark", "starting activation at %s\n", url)
}

func init() {
	activateCmd.Flags().StringVarP(&activateDeployKey, "deploy-key", "k", "", fmt.Sprintf("the deployment key to activate"))
	//rootCmd.AddCommand(activateCmd)
}
