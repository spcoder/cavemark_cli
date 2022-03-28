package cmd

import (
	"embed"
	"github.com/spf13/cobra"
	"io/fs"
	"io/ioutil"
	"os"
	"strings"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "initialize a project",
	Long: `Initializes a Cavemark project.

Example:
  # initializes a Cavemark project, with common defaults, in the current directory.
  cavemark init`,
	Args: cobra.MaximumNArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		printInitHeader(cmd.Parent().Version)
		return replicateFS()
	},
}

func printInitHeader(version string) {
	p("cavemark", "version %s\n", version)
	p("cavemark", "initializing project\n")
}

//go:embed _init/*
var initFS embed.FS

func replicateFS() error {
	rootDir := "_init"
	return fs.WalkDir(initFS, rootDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == rootDir {
			return nil
		}

		filePath := strings.Replace(path, rootDir+"/", "", 1)

		if d.IsDir() {
			p("cavemark", "creating directory: %s\n", filePath)
			err := os.Mkdir(filePath, os.FileMode(0755))
			if err != nil {
				return err
			}
		} else {
			p("cavemark", "creating file: %s\n", filePath)
			data, err := initFS.ReadFile(path)
			if err != nil {
				return err
			}
			err = ioutil.WriteFile(filePath, data, os.FileMode(0755))
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func init() {
	rootCmd.AddCommand(initCmd)
}
