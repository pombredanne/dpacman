package dpacman

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/dotcloud/docker/archive"
)

const (
	BUILDER_INPROGRESS_FOLDER = "/var/lib/dpacman/inprogress"
	BUILDER_SUCCESSFUL_FOLDER = "/var/lib/dpacman/successful"
	BUILDER_ERROR_FOLDER      = "/var/lib/dpacman/error"
)

func (in *Installer) BuildPackage(src_path string) (string, error) {
	if err := prepareBuilderEnv(); err != nil {
		log.Print("Error preparing builder env")
		return "", err
	}

	src_abs_path, err := filepath.Abs(src_path)
	if err != nil {
		return "", errors.New("Error determining package's absolute path: " + err.Error())
	}

	dpacman_file := filepath.Join(src_abs_path, PACKAGE_SPEC_FILE)
	p, err := LoadPackageSpec(dpacman_file)
	if err != nil {
		return "", errors.New("Can't load Dpacman: " + err.Error())
	}

	error_folder, err := createErrorFolder(p.FullName())
	if err != nil {
		return "", err
	}

	inprogress_folder, err := createInprogressFolder(p.FullName())
	if err != nil {
		return "", err
	}

	if err := copyContent(src_abs_path, inprogress_folder); err != nil {
		return "", err
	}

	p.Path = inprogress_folder

	// Check all defined files in Dpacman, exists in the package's folder
	log.Println("Checking defined files...")
	if err := p.CheckFilesExist(); err != nil {
		log.Print("Error checking package's files")

		if err := moveContent(inprogress_folder, error_folder); err != nil {
			return "", err
		}
		os.RemoveAll(inprogress_folder)

		return "", err
	}

	// Save from Docker all images defined in Dpacman
	if err := os.Mkdir(filepath.Join(inprogress_folder, "images"), 0755); err != nil {
		return "", err
	}

	for _, i := range p.Images {
		log.Println("Saving image " + i.FullName())
		if err := in.SaveImage(p, i); err != nil {
			log.Print("Error saving image")

			if err := moveContent(inprogress_folder, error_folder); err != nil {
				return "", err
			}
			os.RemoveAll(inprogress_folder)

			return "", err
		}
	}

	out_filename := filepath.Join(p.FullName() + ".tar.gz")
	out_filepath := filepath.Join(inprogress_folder, out_filename)

	out, err := archive.Tar(inprogress_folder, archive.Gzip)
	if err != nil {
		log.Print("Error compressing " + inprogress_folder + " content")

		if err := moveContent(inprogress_folder, error_folder); err != nil {
			return "", err
		}
		os.RemoveAll(inprogress_folder)

		return "", err
	}

	f, err := os.Create(out_filepath)
	if err != nil {
		log.Print("Error creating output file " + out_filepath)

		if err := moveContent(inprogress_folder, error_folder); err != nil {
			return "", err
		}
		os.RemoveAll(inprogress_folder)

		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, out); err != nil {
		log.Print("Error copying tar content to " + out_filepath)
		f.Close()

		if err := moveContent(inprogress_folder, error_folder); err != nil {
			return "", err
		}
		os.RemoveAll(inprogress_folder)

		return "", err
	}
	f.Close()

	successful_folder, err := createSuccessfulFolder(p.FullName())
	if err != nil {
		return "", err
	}

	if err := moveContent(inprogress_folder, successful_folder); err != nil {
		return "", err
	}
	os.RemoveAll(inprogress_folder)

	return filepath.Join(successful_folder, out_filename), nil
}

func createSuccessfulFolder(name string) (string, error) {
	// Create the tmp folder
	tmp_src_path, err := ioutil.TempDir(BUILDER_SUCCESSFUL_FOLDER, name)
	if err != nil {
		return "", errors.New("Error creating " + tmp_src_path + ": " + err.Error())
	}
	return tmp_src_path, nil
}

func createErrorFolder(name string) (string, error) {
	// Create the tmp folder
	tmp_src_path, err := ioutil.TempDir(BUILDER_ERROR_FOLDER, name)
	if err != nil {
		return "", errors.New("Error creating " + tmp_src_path + ": " + err.Error())
	}
	return tmp_src_path, nil
}

func createInprogressFolder(name string) (string, error) {
	// Create the tmp folder
	tmp_src_path, err := ioutil.TempDir(BUILDER_INPROGRESS_FOLDER, name)
	if err != nil {
		return "", errors.New("Error creating " + tmp_src_path + ": " + err.Error())
	}
	return tmp_src_path, nil
}

func copyContent(src, dst string) error {
	cmd := fmt.Sprintf("cp -r %v %v", filepath.Join(src, "*"), dst)
	c := exec.Command("bash", "-c", cmd)
	if out, err := c.CombinedOutput(); err != nil {
		return errors.New("Error copying content from " + src + " to " + dst + ": " + string(out))
	}
	return nil
}

func moveContent(src, dst string) error {
	cmd := fmt.Sprintf("mv %v %v", filepath.Join(src, "*"), dst)
	c := exec.Command("bash", "-c", cmd)
	if out, err := c.CombinedOutput(); err != nil {
		return errors.New("Error moving content from " + src + " to " + dst + ": " + string(out))
	}
	return nil
}

func prepareBuilderEnv() error {
	folders := []string{BUILDER_ERROR_FOLDER, BUILDER_SUCCESSFUL_FOLDER, BUILDER_INPROGRESS_FOLDER}
	for _, f := range folders {
		c := exec.Command("mkdir", "-p", f)
		if out, err := c.CombinedOutput(); err != nil {
			return errors.New("Error creating " + f + ": " + string(out))
		}
	}
	return nil
}
