package dpacman

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/docker/docker/pkg/archive"
	"gopkg.in/yaml.v1"
)

type Package struct {
	Name        string
	Version     string
	Release     int
	Maintainer  string
	Epoch       string
	Description string
	Changelog   string
	Images      []*Image
	Files       []string
	PreInstall  string
	PostInstall string

	// Local path where the package is decompressed
	Path string
}

const (
	PACKAGE_SPEC_FILE              = "Dpacman"
	INSTALLATION_MARK_CONTENT_TMPL = `name: %v
version: %v
release: %v
epoch: %v
`
	INFO_TMPL = `Package: {{.Name}}
Version: {{.Version}}-{{.Release}}
Maintainer: {{.Maintainer}}
Description: {{.Description}}
Changelog: {{.Changelog}}
`
)

func LoadFromLocalPath(file_path string) (*Package, error) {
	dst, err := ioutil.TempDir("", path.Base(file_path))
	if err != nil {
		return nil, err
	}

	if err := archive.UntarPath(file_path, dst); err != nil {
		return nil, err
	}

	p, err := LoadPackageSpec(filepath.Join(dst, PACKAGE_SPEC_FILE))
	if err != nil {
		return nil, err
	}

	p.Path = dst

	return p, err
}

func LoadPackageSpec(filepath string) (*Package, error) {
	b, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var p *Package
	err = yaml.Unmarshal(b, &p)

	p.Description = strings.TrimSpace(p.Description)
	p.Changelog = strings.TrimSpace(p.Changelog)

	return p, err
}

func (p *Package) FullName() string {
	return fmt.Sprintf("%v-%v-%v", p.Name, p.Version, p.Release)
}

func (p *Package) CheckFilesExist() error {
	for _, f := range p.Files {
		if _, err := os.Stat(filepath.Join(p.Path, "files", f)); err != nil {
			if os.IsNotExist(err) {
				return errors.New(f + "is defined but doesn't exists")
			}
			return err
		}
	}

	return nil
}

func (p *Package) InstallFiles() error {
	if err := p.backupFiles(); err != nil {
		return errors.New("Error making file's backups: " + err.Error())
	}

	for _, f := range p.Files {
		src := filepath.Join(p.Path, "files", f)
		dst := filepath.Join("/", f)

		if err := cpFile(src, dst); err != nil {
			return err
		}
	}

	return nil
}

func (p *Package) DoPre() ([]byte, error) {
	if p.PreInstall == "" {
		return []byte{}, nil
	}

	f, err := ioutil.TempFile("", fmt.Sprintf("%v-%v-%v-preinstall", p.Name, p.Version, p.Release))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	defer os.Remove(f.Name())

	f.WriteString(p.PreInstall)

	c := exec.Command("/bin/bash", f.Name())
	return c.CombinedOutput()
}

func (p *Package) DoPost() ([]byte, error) {
	if p.PostInstall == "" {
		return []byte{}, nil
	}

	f, err := ioutil.TempFile("", fmt.Sprintf("%v-%v-%v-postinstall", p.Name, p.Version, p.Release))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	defer os.Remove(f.Name())

	f.WriteString(p.PostInstall)

	c := exec.Command("/bin/bash", f.Name())
	return c.CombinedOutput()
}

func (p *Package) CreateMark(marks_path string) error {
	content := fmt.Sprintf(INSTALLATION_MARK_CONTENT_TMPL, p.Name, p.Version, p.Release, p.Epoch)

	if err := os.MkdirAll(marks_path, 0755); err != nil {
		return errors.New("Error creating " + path.Dir(marks_path) + " : " + err.Error())
	}

	f, err := os.Create(filepath.Join(marks_path, p.Name+".package"))
	if err != nil {
		return errors.New("Error opening " + marks_path + " : " + err.Error())
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		return errors.New("Error writing mark's content:" + err.Error())
	}

	return nil
}

func (p *Package) Clean() error {
	if err := os.RemoveAll(p.Path); err != nil {
		return err
	}

	return nil
}

func (p *Package) String() (string, error) {
	out := new(bytes.Buffer)
	tmpl, _ := template.New("package-info").Parse(INFO_TMPL)
	if err := tmpl.Execute(out, p); err != nil {
		return "", errors.New("Error printing package info: " + err.Error())
	}

	return out.String(), nil
}

func (p *Package) backupFiles() error {
	for _, f := range p.Files {
		// All files must be children of /
		fpath := path.Join("/", f)

		// Only backup files that exists. Don't fail if doesn't
		if _, err := os.Stat(fpath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}

		if err := cpFile(fpath, fpath+".old"); err != nil {
			return err
		}
	}

	return nil
}

func cpFile(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	if err := os.MkdirAll(path.Dir(dst), 0755); err != nil {
		return err
	}

	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer d.Close()

	if _, err := io.Copy(d, s); err != nil {
		return err
	}

	return nil
}
