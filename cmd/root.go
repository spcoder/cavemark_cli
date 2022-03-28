package cmd

import (
	"fmt"
	"os"

	_ "github.com/joho/godotenv/autoload"
	"github.com/spf13/cobra"
)

var (
	url          string
	apiKey       string
	apiSecretKey string
)

const (
	cavemarkUrl          = "CAVEMARK_URL"
	cavemarkApiKey       = "CAVEMARK_API_KEY"
	cavemarkApiSecretKey = "CAVEMARK_API_SECRET_KEY"
)

var rootCmd = &cobra.Command{
	Use:     "cavemark",
	Version: "1.0.0",
	Short:   "cavemark controls Cavemark instances",
	Long:    `The cavemark command lets you control instances of Cavemark.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&url, "url", "u", "", fmt.Sprintf("the url to Cavemark [%s]", cavemarkUrl))
	rootCmd.PersistentFlags().StringVarP(&apiKey, "api-key", "", "", fmt.Sprintf("the api key [%s]", cavemarkApiKey))
	rootCmd.PersistentFlags().StringVarP(&apiSecretKey, "api-secret-key", "", "", fmt.Sprintf("the api secret key [%s]", cavemarkApiSecretKey))

	if url == "" {
		url = os.Getenv(cavemarkUrl)
		if url == "" {
			url = "https://deploy.cavemark.com"
		}
	}

	if apiKey == "" {
		apiKey = os.Getenv(cavemarkApiKey)
	}

	if apiSecretKey == "" {
		apiSecretKey = os.Getenv(cavemarkApiSecretKey)
	}
}
