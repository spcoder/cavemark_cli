package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sort"
	"time"

	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "retrieves Cavemark information.",
	Long: `Retrieves information about a Cavemark instance.

Example:
  # activates the code running in the 'example' deployment at https://example.com
  cavemark activate -u https://example.com -k example`,
}

var getDeployKeyCmd = &cobra.Command{
	Use:   "deploy-key",
	Short: "returns the currently activated deployment key",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := getDeployKey()
		if err != nil {
			return err
		}
		fmt.Println(id)
		return nil
	},
}

var getDeployListCmd = &cobra.Command{
	Use:   "deploy-list",
	Short: "returns a list of all deployments",
	RunE: func(cmd *cobra.Command, args []string) error {
		return printDeployList()
	},
}

type DeploymentSummary struct {
	DeployKey string    `json:"deployKey"`
	Timestamp time.Time `json:"timestamp"`
	Active    bool      `json:"active"`
}

func printDeployList() error {
	resp, err := httpGet(fmt.Sprintf("%s/deploy/list", url))
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	deploymentSummaryList := make([]DeploymentSummary, 0)
	err = json.Unmarshal(body, &deploymentSummaryList)
	if err != nil {
		return err
	}
	sort.Slice(deploymentSummaryList, func(i, j int) bool {
		return deploymentSummaryList[i].Timestamp.After(deploymentSummaryList[j].Timestamp)
	})
	fmt.Println("Timestamp                    \tStatus\tDeployment Key")
	fmt.Println("-----------------------------\t------\t--------------")
	for _, ds := range deploymentSummaryList {
		active := ""
		if ds.Active {
			active = "Active"
		}
		fmt.Printf("%s\t%s\t%s\n", ds.Timestamp.Format(time.RFC1123), active, ds.DeployKey)
	}
	return nil
}

func init() {
	getCmd.AddCommand(getDeployKeyCmd)
	getCmd.AddCommand(getDeployListCmd)
	rootCmd.AddCommand(getCmd)
}
