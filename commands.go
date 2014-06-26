package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/codegangsta/cli"
)

var installCmd = cli.Command{
	Name:        "install",
	Usage:       "dpacman build </path/to/dpackage.tar.gz>",
	Description: "Install a Dpackage",
	Action: func(c *cli.Context) {
		if len(c.Args()) == 0 {
			fmt.Println("No package provided!")
			os.Exit(1)
		}

		ppath, err := filepath.Abs(c.Args()[0])
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

		p, err := LoadFromLocalPath(ppath)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

		i, err := NewInstaller(c.GlobalString("docker"))
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

		err = i.InstallPackage(p)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	},
}

var infoCmd = cli.Command{
	Name:        "info",
	Usage:       "dpacman info </path/to/dpackage.tar.gz>",
	Description: "Show info from a Dpackage",
	Action: func(c *cli.Context) {
		if len(c.Args()) == 0 {
			log.Fatal("No package provided!")
		}

		ppath, err := filepath.Abs(c.Args()[0])
		if err != nil {
			log.Fatal(err)
		}

		p, err := LoadFromLocalPath(ppath)
		if err != nil {
			log.Fatal(err)
		}
		defer p.Clean()

		s, err := p.String()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Print(s)
	},
}

var buildCmd = cli.Command{
	Name:        "build",
	Usage:       "dpacman build </path/to/package/contents>",
	Description: "Build a dpacman package from a source folder",
	Action: func(c *cli.Context) {
		if len(c.Args()) == 0 {
			log.Fatal("No source path provided!")
		}

		i, err := NewInstaller(c.GlobalString("docker"))
		if err != nil {
			log.Fatal(err)
		}

		out, err := i.BuildPackage(c.Args()[0])
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Package correctly generated on %v !\n", out)
	},
}
