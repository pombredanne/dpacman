package main

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"

	"github.com/codegangsta/cli"
	"github.com/teambox/dpacman/lib"
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

		p, err := dpacman.LoadFromLocalPath(ppath)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

		i, err := dpacman.NewInstaller(c.GlobalString("docker"))
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

		p, err := dpacman.LoadFromLocalPath(ppath)
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
	Usage:       "dpacman build </path/to/package/Dpacman>",
	Description: "Build a dpacman package from a source folder",
	Action: func(c *cli.Context) {
		if len(c.Args()) == 0 {
			log.Fatal("No source path provided!")
		}

		u, err := user.Current()
		if err != nil {
			log.Fatal(err)
		}
		builder_path := filepath.Join(u.HomeDir, ".dpacman")

		b, err := dpacman.NewBuilder(c.GlobalString("docker"), builder_path)
		if err != nil {
			log.Fatal(err)
		}

		out, err := b.BuildPackage(c.Args()[0])
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Package correctly generated on %v !\n", out)
	},
}
