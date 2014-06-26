package main

type Image struct {
	Tag  string
	Repo string
	Path string
}

func (i *Image) FullName() string {
	return i.Repo + ":" + i.Tag
}
