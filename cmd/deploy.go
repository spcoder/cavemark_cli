package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"

	"github.com/spf13/cobra"

	"github.com/evanw/esbuild/pkg/api"
)

var (
	funcDir            string
	staticDir          string
	defaultStaticDir   = "static"
	resourceDir        string
	defaultResourceDir = "resource"
	strategy           string
	manualDeployKey    string
	watch              bool
)

const (
	cavemarkFuncDir     = "CAVEMARK_FUNC_DIR"
	cavemarkStaticDir   = "CAVEMARK_STATIC_DIR"
	cavemarkResourceDir = "CAVEMARK_RESOURCE_DIR"
	cavemarkStrategy    = "CAVEMARK_STRATEGY"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "deploy to Cavemark",
	Long: `Deploy to Cavemark.

Strategies:
When deploying you'll need to choose a strategy.
* bluegreen = rotates between blue and green deployments
* manual    = you supply the deployment key

Secrets:
Any environment variable that starts with CAVEMARK_ will be deployed to Cavemark as secrets.
Secrets will be available to Cavemark functions without the CAVEMARK_SECRET. For example,
CAVEMARK_SECRET_PG_CONNECTION will be available as PG_CONNECTION.

Examples:
  # deploys all *.js files recursively in the "src" directory to http://localhost:9090 using the bluegreen strategy
  cavemark deploy

  # deploys all *.js files recursively in ~/dev/project/server to https://example.com using the word 'example'' as the deployment key
  # also, deploys all files (except hidden) in ~/dev/project/assets to https://example.com as static assets using the same strategy
  cavemark deploy -f ~/dev/project/server -s ~/dev/project/assets -u https://example.com -g manual -k example`,
	Args: cobra.MaximumNArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		printDeployHeader(cmd.Parent().Version)

		err := validate()
		if err != nil {
			return err
		}

		strategyFunc, err := resolveStrategy()
		if err != nil {
			return err
		}

		err = strategyFunc()
		if err != nil {
			p("error", "%s\n", err)
			return err
		}

		if watch {
			err = startWatching(strategyFunc)
			if err != nil {
				return err
			}
		}

		return nil
	},
}

func validate() error {
	if url == "" {
		return errors.New("url is required")
	}
	if apiKey == "" {
		return errors.New("api key is required")
	}
	if apiSecretKey == "" {
		return errors.New("api secret key is required")
	}
	indexExists, err := indexFunctionExists()
	if err != nil {
		return err
	}
	staticsExist, err := staticFilesExist()
	if err != nil {
		return err
	}
	if !indexExists && !staticsExist {
		return errors.New("no index.js or static files to deploy")
	}
	return nil
}

func indexFunctionExists() (bool, error) {
	_, err := os.Lstat(path.Join(funcDir, "index.js"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func staticFilesExist() (bool, error) {
	if staticDir == "" {
		return false, nil
	}
	_, err := os.Lstat(staticDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	files, err := globAll(staticDir)
	if err != nil {
		return false, err
	}
	if len(files) == 0 {
		return false, nil
	}
	return true, nil
}

type strategyFunction func() error

func resolveStrategy() (strategyFunction, error) {
	switch strategy {
	case "bluegreen":
		return bluegreen, nil
	case "manual":
		if manualDeployKey == "" {
			return nil, fmt.Errorf("please supply the deploy-key paramter")
		}
		return manual, nil
	default:
		return nil, fmt.Errorf("strategy (%s) not supported", strategy)
	}
}

func printDeployHeader(version string) {
	p("cavemark", "version %s\n", version)
	p("cavemark", "starting deployment to %s\n", url)
	p("strategy", "using strategy %s\n", strategy)
}

func p(key, msg string, args ...interface{}) {
	if key == "" {
		fmt.Printf(msg, args...)
		return
	}
	fmt.Printf("%10s: %s", strings.ToUpper(key), fmt.Sprintf(msg, args...))
}

func startWatching(fn strategyFunction) error {
	p("watch", "starting to watch directories for changes\n")
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer func() { _ = watcher.Close() }()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					fmt.Printf("\n\n")
					p("watch", "detected file system change\n")
					err = fn()
					if err != nil {
						p("error", "%s\n", err)
					}
					fmt.Print("\n\n\a")
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				p("error", "%s\n", err)
			}
		}
	}()

	err = filepath.WalkDir(funcDir, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			p("watch", path+"\n")
			return watcher.Add(path)
		}
		return nil
	})
	if resourceDir != "" {
		err = filepath.WalkDir(resourceDir, func(path string, d fs.DirEntry, err error) error {
			if d.IsDir() {
				p("watch", path+"\n")
				return watcher.Add(path)
			}
			return nil
		})
	}
	if staticDir != "" {
		err = filepath.WalkDir(staticDir, func(path string, d fs.DirEntry, err error) error {
			if d.IsDir() {
				p("watch", path+"\n")
				return watcher.Add(path)
			}
			return nil
		})
	}
	if err != nil {
		return err
	}
	<-done

	return nil
}

func httpCall(method, url, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	if apiKey != "" {
		req.Header.Set("API_KEY", apiKey)
	}
	if apiSecretKey != "" {
		req.Header.Set("API_SECRET_KEY", apiSecretKey)
	}
	client := &http.Client{}
	return client.Do(req)
}

func httpPut(url, contentType string, body io.Reader) (*http.Response, error) {
	return httpCall(http.MethodPut, url, contentType, body)
}

func httpPost(url, contentType string, body io.Reader) (*http.Response, error) {
	return httpCall(http.MethodPost, url, contentType, body)
}

func httpGet(url string) (*http.Response, error) {
	return httpCall(http.MethodGet, url, "text/plain", nil)
}

func bluegreen() error {
	deployKey, err := getDeployKey()
	if err != nil {
		return fmt.Errorf("error getting deploy key: %w", err)
	}
	if deployKey == "blue" {
		deployKey = "green"
	} else {
		deployKey = "blue"
	}
	return deploy(deployKey)
}

func manual() error {
	return deploy(manualDeployKey)
}

func deploy(deployKey string) error {
	p(strategy, "deploying to %s\n", deployKey)
	err := beginDeployment(deployKey)
	if err != nil {
		return err
	}
	err = deploySecrets(deployKey)
	if err != nil {
		return err
	}
	err = deployFunction(deployKey)
	if err != nil {
		return err
	}
	err = deployResources(deployKey)
	if err != nil {
		return err
	}
	err = deployStatics(deployKey)
	if err != nil {
		return err
	}
	err = activateDeployment(deployKey)
	if err != nil {
		return err
	}
	return nil
}

func getDeployKey() (string, error) {
	resp, err := httpGet(fmt.Sprintf("%s/cvmrk/cli/deploy", url))
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func deploySecrets(deployKey string) error {
	p("secrets", "starting to deploy secrets\n")
	for _, v := range envSecrets() {
		p("secrets", "deploying %s", v.key)
		resp, err := httpPut(fmt.Sprintf("%s/cvmrk/cli/deploy/%s/secret/%s", url, deployKey, v.key), "text/plain", strings.NewReader(v.value))
		if err != nil {
			return fmt.Errorf("error deploying secret (%s): %w", v.key, err)
		}
		if resp.StatusCode == http.StatusNoContent {
			p("", " [OK]\n")
		} else {
			p("", " [%d]\n", resp.StatusCode)
			return fmt.Errorf("failed to deploy secret (%s)", v.key)
		}
	}
	p("secrets", "successfully deployed\n")
	return nil
}

func bundle() ([]byte, error) {
	entryFile := path.Join(funcDir, "index.js")
	result := api.Build(api.BuildOptions{
		Bundle:           true,
		MinifySyntax:     true,
		MinifyWhitespace: true,
		Color:            api.ColorNever,
		TreeShaking:      api.TreeShakingFalse,
		EntryPoints:      []string{entryFile},
		Platform:         api.PlatformNode,
		LogLevel:         api.LogLevelInfo,
	})
	if len(result.Errors) > 0 {
		for _, err := range result.Errors {
			fmt.Println(err.Text)
		}
		return nil, errors.New("error while bundling")
	}
	return result.OutputFiles[0].Contents, nil
}

func deployFunction(deployKey string) error {
	indexExists, err := indexFunctionExists()
	if err != nil {
		return err
	}
	if !indexExists {
		return nil
	}
	p("functions", "starting to deploy functions in '%s'\n", funcDir)
	p("functions", "creating bundle")
	content, err := bundle()
	if err != nil {
		return err
	}
	p("", " [OK]\n")

	p("functions", "deploying bundle")
	resp, err := httpPut(fmt.Sprintf("%s/cvmrk/cli/deploy/%s/function", url, deployKey), "text/plain", bytes.NewReader(content))
	if err != nil {
		return fmt.Errorf("error deploying bundle: %w", err)
	}
	if resp.StatusCode == http.StatusNoContent {
		p("", " [OK]\n")
	} else {
		p("", " [%d]\n", resp.StatusCode)
		return fmt.Errorf("failed to deploy bundle")
	}
	p("functions", "successfully deployed\n")
	return nil
}

func deployResources(deployKey string) error {
	if resourceDir == "" {
		return nil
	}
	_, err := os.Lstat(resourceDir)
	if err != nil {
		if strings.HasSuffix(err.Error(), "no such file or directory") && resourceDir == defaultResourceDir {
			return nil
		}
		return err
	}
	p("resources", "starting to deploy resource files in '%s'\n", resourceDir)
	files, err := globAll(resourceDir)
	if err != nil {
		return fmt.Errorf("error globbing files: %w", err)
	}
	for _, f := range files {
		p("resources", "deploying file %s", f)
		contents, err := ioutil.ReadFile(f)
		if err != nil {
			return fmt.Errorf("error reading file (%s): %w", f, err)
		}
		filePath := filepath.ToSlash(removeDir(f, resourceDir))
		contentType := http.DetectContentType(contents)
		resp, err := httpPut(fmt.Sprintf("%s/cvmrk/cli/deploy/%s/resource/%s", url, deployKey, filePath), contentType, bytes.NewReader(contents))
		if err != nil {
			return fmt.Errorf("error deploying resource file (%s): %w", f, err)
		}
		if resp.StatusCode == http.StatusNoContent {
			p("", " [OK]\n")
		} else {
			p("", " [%d]\n", resp.StatusCode)
			return fmt.Errorf("failed to deploy resource file (%s)", f)
		}
	}
	p("resources", "successfully deployed\n")
	return nil
}

func deployStatics(deployKey string) error {
	if staticDir == "" {
		return nil
	}
	_, err := os.Lstat(staticDir)
	if err != nil {
		if strings.HasSuffix(err.Error(), "no such file or directory") && staticDir == defaultStaticDir {
			return nil
		}
		return err
	}
	p("statics", "starting to deploy static files in '%s'\n", staticDir)
	files, err := globAll(staticDir)
	if err != nil {
		return fmt.Errorf("error globbing files: %w", err)
	}
	for _, f := range files {
		p("statics", "deploying file %s", f)
		contents, err := ioutil.ReadFile(f)
		if err != nil {
			return fmt.Errorf("error reading file (%s): %w", f, err)
		}
		filePath := filepath.ToSlash(removeDir(f, staticDir))
		contentType := http.DetectContentType(contents)
		resp, err := httpPut(fmt.Sprintf("%s/cvmrk/cli/deploy/%s/static/%s", url, deployKey, filePath), contentType, bytes.NewReader(contents))
		if err != nil {
			return fmt.Errorf("error deploying static file (%s): %w", f, err)
		}
		if resp.StatusCode == http.StatusNoContent {
			p("", " [OK]\n")
		} else {
			p("", " [%d]\n", resp.StatusCode)
			return fmt.Errorf("failed to deploy static file (%s)", f)
		}
	}
	p("statics", "successfully deployed\n")
	return nil
}

func removeDir(f, dir string) string {
	s := strings.Replace(f, dir, "", 1)
	if strings.HasPrefix(s, "/") {
		return s[1:]
	}
	return s
}

func beginDeployment(deployKey string) error {
	p("begin", "starting deployment %s\n", deployKey)
	resp, err := httpPost(fmt.Sprintf("%s/cvmrk/cli/deploy/%s/begin", url, deployKey), "text/plain", nil)
	if err != nil {
		return fmt.Errorf("error starting deployment: %w", err)
	}
	if resp.StatusCode == http.StatusNoContent {
		p("begin", "successfully started deployment %s\n", deployKey)
	} else {
		return fmt.Errorf("failed to begin deployment: status code = %d", resp.StatusCode)
	}
	return nil
}

func activateDeployment(deployKey string) error {
	p("activate", "starting to activate %s\n", deployKey)
	resp, err := httpPost(fmt.Sprintf("%s/cvmrk/cli/deploy/%s/activate", url, deployKey), "text/plain", nil)
	if err != nil {
		return fmt.Errorf("error activating deployment: %w", err)
	}
	if resp.StatusCode == http.StatusNoContent {
		p("activate", "successfully activated %s\n", deployKey)
	} else {
		return fmt.Errorf("failed to activate deployment: status code = %d", resp.StatusCode)
	}
	return nil
}

func globAll(dir string) ([]string, error) {
	files := make([]string, 0)
	err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		if strings.HasPrefix(filepath.Base(path), ".") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files, err
}

type envSecret struct {
	key   string
	value string
}

func envSecrets() []envSecret {
	result := make([]envSecret, 0)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if strings.HasPrefix(pair[0], "CAVEMARK_SECRET_") {
			key := strings.Replace(pair[0], "CAVEMARK_SECRET_", "", 1)
			result = append(result, envSecret{key, pair[1]})
		}
	}
	return result
}

func resolveStringFlag(value, envVar, fallback string) string {
	if value == "" {
		value = os.Getenv(envVar)
	}
	if value == "" {
		return fallback
	}
	return value
}

func init() {
	deployCmd.Flags().StringVarP(&funcDir, "func-dir", "f", "", fmt.Sprintf("the directory that contains functions to deploy [%s]", cavemarkFuncDir))
	deployCmd.Flags().StringVarP(&resourceDir, "resource-dir", "r", "", fmt.Sprintf("the directory that contains resource files to deploy [%s]", cavemarkResourceDir))
	deployCmd.Flags().StringVarP(&staticDir, "static-dir", "s", "", fmt.Sprintf("the directory that contains static assets to deploy [%s]", cavemarkStaticDir))
	deployCmd.Flags().StringVarP(&strategy, "strategy", "g", "", fmt.Sprintf("the deployment strategy (bluegreen, manual) [%s]", cavemarkStrategy))
	deployCmd.Flags().StringVarP(&manualDeployKey, "deploy-key", "k", "", fmt.Sprintf("a manually specified deployment key, should not be used with strategy"))
	deployCmd.Flags().BoolVarP(&watch, "watch", "w", false, "deploy when directory changes")
	rootCmd.AddCommand(deployCmd)

	funcDir = resolveStringFlag(funcDir, cavemarkFuncDir, "src")
	resourceDir = resolveStringFlag(resourceDir, cavemarkResourceDir, defaultResourceDir)
	staticDir = resolveStringFlag(staticDir, cavemarkStaticDir, defaultStaticDir)
	strategy = resolveStringFlag(strategy, cavemarkStrategy, "bluegreen")
}
