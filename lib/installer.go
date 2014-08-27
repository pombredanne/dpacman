package dpacman

import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"

	"github.com/fsouza/go-dockerclient"
)

const (
	INSTALLATION_MARKS_PATH = "/etc/dpacman/"
)

type Installer struct {
	DockerClient *docker.Client
}

func NewInstaller(endpoint string) (*Installer, error) {
	client, err := docker.NewClient(endpoint)
	if err != nil {
		return nil, err
	}

	return &Installer{
		DockerClient: client,
	}, nil
}

func (in *Installer) InstallPackage(p *Package) error {
	log.Print("Running pre-install script...")
	out, err := p.DoPre()
	if err != nil {
		log.Print(string(out))
		return errors.New("Error running the pre-installation step: " + err.Error())
	}

	log.Print("Importing images")
	for _, img := range p.Images {
		log.Printf("Imporing image %s:%s...", img.Repo, img.Tag)
		err := in.ImportImage(p, img)
		if err != nil {
			return errors.New("Error importing image " + img.Repo + " : " + err.Error())
		}
	}

	log.Print("Installing files...")
	err = p.InstallFiles()
	if err != nil {
		return errors.New("Error installing files: " + err.Error())
	}

	log.Print("Running post-install script...")
	out, err = p.DoPost()
	if err != nil {
		log.Print(string(out))
		return errors.New("Error running the post-installation step: " + err.Error())
	}

	log.Print("Creating installation mark...")
	err = p.CreateMark(INSTALLATION_MARKS_PATH)
	if err != nil {
		return errors.New("Error creating the installation mark: " + err.Error())
	}

	log.Print("Cleaning package's tmp folder...")
	err = p.Clean()
	if err != nil {
		return errors.New("Error cleaning package's tmp folder : " + err.Error())
	}

	return nil
}

func (in *Installer) ImportImage(p *Package, img *Image) error {
	cmd := fmt.Sprintf("docker load -i %v", filepath.Join(p.Path, img.Path))
	c := exec.Command("bash", "-c", cmd)
	if out, err := c.CombinedOutput(); err != nil {
		return errors.New("Error loading " + img.FullName() + ": " + string(out))
	}

	return nil
}
