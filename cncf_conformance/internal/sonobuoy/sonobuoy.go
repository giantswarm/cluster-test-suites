package sonobuoy

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"

	"sigs.k8s.io/yaml"

	"github.com/giantswarm/clustertest/pkg/logger"
)

type Client struct {
	kubeconfig string
	results    results
}

func New(kubeconfigPath string) *Client {
	return &Client{
		kubeconfig: kubeconfigPath,
	}
}

func (c *Client) BinaryExists() bool {
	path, err := exec.LookPath("sonobuoy")
	return err == nil && path != ""
}

func (c *Client) RunTests(mode string) error {
	cmd, err := c.cmd("run", "--wait", "--mode", mode)
	if err != nil {
		return err
	}
	return cmd.Run()
}

func (c *Client) SaveResults() error {
	cmd, err := c.cmd("retrieve", os.Getenv("RESULTS_DIRECTORY"), "--extract")
	if err != nil {
		return err
	}
	err = cmd.Run()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path.Join(os.Getenv("RESULTS_DIRECTORY"), "plugins/e2e", "sonobuoy_results.yaml"))
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, &c.results)
}

func (c *Client) HasPassed() bool {
	return c.results.Status == "passed"
}

func (c *Client) GetFailed() []string {
	const failedStatus = "failed"

	failedTests := []string{}
	if c.results.Items != nil {
		for _, report := range c.results.Items {
			if report.Status == failedStatus && report.Items != nil {
				for _, plugin := range report.Items {
					if plugin.Status == failedStatus && plugin.Items != nil {
						for _, test := range plugin.Items {
							if test.Status == failedStatus {
								failedTests = append(failedTests, test.Name)
							}
						}
					}
				}
			}
		}
	}
	return failedTests
}

func (c *Client) cmd(args ...string) (*exec.Cmd, error) {
	cmd := exec.Command("sonobuoy", args...)

	cmd.Env = append(cmd.Env, fmt.Sprintf("KUBECONFIG=%s", c.kubeconfig))

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	go io.Copy(logger.LogWriter, stdout)
	go io.Copy(logger.LogWriter, stderr)

	return cmd, nil
}
