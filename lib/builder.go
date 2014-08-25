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
	"github.com/fsouza/go-dockerclient"
)

type Builder struct {
	DockerClient           *docker.Client
	BuilderRootFolder      string
	successfulBuildsFolder string
	inprogressBuildsFolder string
	failedBuildsFolder     string
}

func NewBuilder(endpoint string, builder_root_folder string) (*Builder, error) {
	client, err := docker.NewClient(endpoint)
	if err != nil {
		return nil, err
	}

	return &Builder{
		DockerClient:           client,
		BuilderRootFolder:      builder_root_folder,
		successfulBuildsFolder: filepath.Join(builder_root_folder, "/successful"),
		inprogressBuildsFolder: filepath.Join(builder_root_folder, "/inprogress"),
		failedBuildsFolder:     filepath.Join(builder_root_folder, "/failed"),
	}, nil
}

func (b *Builder) BuildPackage(src_path string) (string, error) {

	p, err := LoadPackageSpec(src_path)
	if err != nil {
		return "", errors.New("Can't load Dpacman: " + err.Error())
	}

	if err := b.prepareBuilderEnv(); err != nil {
		log.Print("Error preparing builder env")
		return "", err
	}

	failed_folder, err := b.createFailedFolder(p.FullName())
	if err != nil {
		return "", err
	}

	inprogress_folder, err := b.createInprogressFolder(p.FullName())
	if err != nil {
		return "", err
	}

	src_abs_path, err := filepath.Abs(src_path)
	if err != nil {
		return "", errors.New("Error determining package's absolute path: " + err.Error())
	}

	if err := copyContent(filepath.Dir(src_abs_path), inprogress_folder); err != nil {
		return "", err
	}

	p.Path = inprogress_folder

	// Check all defined files in Dpacman, exists in the package's folder
	log.Println("Checking defined files...")
	if err := p.CheckFilesExist(); err != nil {
		log.Print("Error checking package's files")

		if err := moveContent(inprogress_folder, failed_folder); err != nil {
			return "", err
		}

		if err := b.createFailedBuildLink(failed_folder); err != nil {
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
		if err := b.SaveImage(p, i); err != nil {
			log.Print("Error saving image")

			if err := moveContent(inprogress_folder, failed_folder); err != nil {
				return "", err
			}

			if err := b.createFailedBuildLink(failed_folder); err != nil {
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

		if err := moveContent(inprogress_folder, failed_folder); err != nil {
			return "", err
		}

		if err := b.createFailedBuildLink(failed_folder); err != nil {
			return "", err
		}

		os.RemoveAll(inprogress_folder)

		return "", err
	}

	f, err := os.Create(out_filepath)
	if err != nil {
		log.Print("Error creating output file " + out_filepath)

		if err := moveContent(inprogress_folder, failed_folder); err != nil {
			return "", err
		}

		if err := b.createFailedBuildLink(failed_folder); err != nil {
			return "", err
		}

		os.RemoveAll(inprogress_folder)

		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, out); err != nil {
		log.Print("Error copying tar content to " + out_filepath)
		f.Close()

		if err := moveContent(inprogress_folder, failed_folder); err != nil {
			return "", err
		}

		if err := b.createFailedBuildLink(failed_folder); err != nil {
			return "", err
		}

		os.RemoveAll(inprogress_folder)

		return "", err
	}
	f.Close()

	successful_folder, err := b.createSuccessfulFolder(p.FullName())
	if err != nil {
		return "", err
	}

	if err := moveContent(inprogress_folder, successful_folder); err != nil {
		return "", err
	}

	if err := b.createSuccessfulBuildLink(successful_folder); err != nil {
		return "", err
	}

	os.RemoveAll(inprogress_folder)

	return filepath.Join(successful_folder, out_filename), nil
}

func (b *Builder) createSuccessfulFolder(name string) (string, error) {
	// Create the tmp folder for a successful job
	tmp_src_path, err := ioutil.TempDir(b.successfulBuildsFolder, name)
	if err != nil {
		return "", errors.New("Error creating " + tmp_src_path + ": " + err.Error())
	}
	return tmp_src_path, nil
}

func (b *Builder) createSuccessfulBuildLink(build_path string) error {
	symlink_path := filepath.Join(b.successfulBuildsFolder, "/latest")
	return createSymLink(build_path, symlink_path)
}

func (b *Builder) createFailedFolder(name string) (string, error) {
	// Create the tmp folder for a failed job
	tmp_src_path, err := ioutil.TempDir(b.failedBuildsFolder, name)
	if err != nil {
		return "", errors.New("Error creating " + tmp_src_path + ": " + err.Error())
	}
	return tmp_src_path, nil
}

func (b *Builder) createFailedBuildLink(build_path string) error {
	symlink_path := filepath.Join(b.failedBuildsFolder, "/latest")
	return createSymLink(build_path, symlink_path)
}

func (b *Builder) createInprogressFolder(name string) (string, error) {
	// Create the tmp folder for an inprogress job
	tmp_src_path, err := ioutil.TempDir(b.inprogressBuildsFolder, name)
	if err != nil {
		return "", errors.New("Error creating " + tmp_src_path + ": " + err.Error())
	}
	return tmp_src_path, nil
}

func (b *Builder) prepareBuilderEnv() error {
	folders := []string{b.failedBuildsFolder, b.successfulBuildsFolder, b.inprogressBuildsFolder}
	for _, f := range folders {
		c := exec.Command("mkdir", "-p", f)
		if out, err := c.CombinedOutput(); err != nil {
			return errors.New("Error creating " + f + ": " + string(out))
		}
	}
	return nil
}

func (b *Builder) SaveImage(p *Package, img *Image) error {
	// Export container as an image
	cmd := fmt.Sprintf("docker save %v > %v", img.FullName(), filepath.Join(p.Path, img.Path))
	c := exec.Command("bash", "-c", cmd)
	if out, err := c.CombinedOutput(); err != nil {
		return errors.New("Error saving " + img.FullName() + ": " + string(out))
	}

	return nil
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

func createSymLink(oldname, newname string) error {
	_, err := os.Lstat(newname)

	// If file exists but Lstat raised an error, exit
	if err != nil && !os.IsNotExist(err) {
		log.Print("AHA")
		log.Print(err)
		return err
	}

	os.Remove(newname)
	if err := os.Symlink(oldname, newname); err != nil {
		return errors.New("Error creating " + newname + ": " + err.Error())
	}

	return nil
}
