package main

import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"

	"github.com/fsouza/go-dockerclient"
)

const InstallationMarksPath = "/etc/dpacman/"

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
	err = p.CreateMark(InstallationMarksPath)
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
	opts := docker.ImportImageOptions{
		Source:     filepath.Join(p.Path, img.Path),
		Repository: img.Repo,
		Tag:        img.Tag,
	}

	return in.DockerClient.ImportImage(opts)
}

// This method will use a shrinked image with no history
// - Create container
// - Export container as an image
// - Delete container
func (in *Installer) SaveImage(p *Package, img *Image) error {

	// Create a container based on the provided image
	copts := docker.CreateContainerOptions{
		Config: &docker.Config{
			Cmd:   []string{"/bin/bash"},
			Image: img.FullName(),
		},
	}

	container, err := in.DockerClient.CreateContainer(copts)
	if err != nil {
		return err
	}

	// Export container as an image
	cmd := fmt.Sprintf("docker export %v > %v", container.ID, filepath.Join(p.Path, img.Path))
	c := exec.Command("bash", "-c", cmd)
	if out, err := c.CombinedOutput(); err != nil {
		return errors.New("Error saving " + img.FullName() + ": " + string(out))
	}

	ropts := docker.RemoveContainerOptions{
		ID:    container.ID,
		Force: true,
	}

	if err = in.DockerClient.RemoveContainer(ropts); err != nil {
		return err
	}

	return nil
}
