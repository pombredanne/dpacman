package main

import (
	"os"

	"github.com/codegangsta/cli"
)

const (
	APP_VER           string = "0.0.1"
	DOCKER_ENDPOINT   string = "unix://var/run/docker.sock"
	BUILD_OUTPUT_PATH string = "/var/lib/dpacman/builds"
)

func main() {
	app := cli.NewApp()
	app.Name = "Dpacman"
	app.Usage = "Package manager for Docker-based applications"
	app.Author = "Salvador Girones <salvador@redbooth.com>"
	app.Version = APP_VER
	app.Flags = []cli.Flag{
		cli.StringFlag{"docker", DOCKER_ENDPOINT, "Docker endpoint"},
	}
	app.Commands = []cli.Command{
		installCmd,
		infoCmd,
		buildCmd,
	}
	app.Run(os.Args)

}
