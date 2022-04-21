package cmd

import (
	"embed"
	"fmt"
	"log"
	"runtime"

	"github.com/urfave/cli/v2"
	"sigs.k8s.io/yaml"
)

type Build struct {
	Version string
	Branch  string
	Commit  string
	Date    string
}

//go:embed build-info.yaml
var versionFs embed.FS

var BuildInfo = Build{}

func init() {
	Commands = append(Commands, BuildInfoCmd)

	buildInfoBytes, err := versionFs.ReadFile("build-info.yaml")
	if err != nil {
		log.Printf("failed to read 'cmd/version.yaml', perhaps it has not been embedded")
	}

	if err := yaml.Unmarshal(buildInfoBytes, &BuildInfo); err != nil {
		log.Printf("failed to read 'cmd/version.yaml': %s", err)
	}
}

var BuildInfoCmd = &cli.Command{
	Name:        "build-info",
	Aliases:     []string{},
	Description: "show version and build info",
	Usage:       "show version and build info",
	ArgsUsage:   "",
	Action:      BuildInfoMain,

	Flags: []cli.Flag{},
}

func BuildInfoMain(c *cli.Context) error {
	fmt.Printf("Client version: %s\n", BuildInfo.Version)
	fmt.Printf("Go version (client): %s\n", runtime.Version())
	fmt.Printf("Build date (client): %s\n", BuildInfo.Date)
	fmt.Printf("Git branch (client): %s\n", BuildInfo.Branch)
	fmt.Printf("Git commit (client): %s\n", BuildInfo.Commit)
	fmt.Printf("OS/Arch (client): %s/%s\n", runtime.GOOS, runtime.GOARCH)

	return nil
}
