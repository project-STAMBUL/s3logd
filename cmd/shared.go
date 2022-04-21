package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

// Any cli command can be appended to this during init() so apps can just add
// all of the available commands as sub-commands.
var Commands = make([]*cli.Command, 0)

// Checks for a minimum number of args required. Using it is as follows.
//   if err := NArgsRequired(c, 1); err != nil {
//   	return err
//   }
func NArgsRequired(c *cli.Context, n int) error {
	if c.NArg() < n {
		exitstr := fmt.Sprintf("command requires at least %d arguments (perhaps check out --help)", n)
		return cli.Exit(exitstr, 1)
	}
	return nil
}

// Build kubernetes config from kubeconfig flag or not, whatever works.
func buildConfig(kubeconfig string) (*rest.Config, error) {
	if cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig); err == nil {
		return cfg, nil
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func DefaultAnnounceAddr() string {
	hostname, err := os.Hostname()
	if err != nil {
		klog.Errorf("%s", err)
	}
	return strings.Join([]string{hostname, "4000"}, ":")
}
