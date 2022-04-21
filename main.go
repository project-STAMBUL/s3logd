package main

import (
	"os"

	"github.com/protosam/s3logd/cmd"
	"github.com/urfave/cli/v2"
)

//go:generate ./scripts/gen-build-info.sh

var App = &cli.App{
	Name:    "s3logd",
	Version: cmd.BuildInfo.Version,
	Authors: []*cli.Author{
		{
			Name:  "protosam",
			Email: "github.com/protosam",
		},
	},
	Copyright: "(c) github.com/protosam",
	HelpName:  "s3logd",
	Usage:     "pushes log data to object storage with S3 compatible API",
	Commands:  cmd.Commands,
}

func main() {
	App.Run(os.Args)
}
