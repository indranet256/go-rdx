package rdx

import "os"

type Repo struct {
	index  map[ID]int
	revlog os.File
}

var RepoFileName = "REPO"
var KeyFileExt = ".key"

func (repo *Repo) New(path string) (err error) {
	return nil
}

func (repo *Repo) NewBranch() (src uint64, err error) {
	return
}

func (repo *Repo) Open(path string) (err error) {
	return nil

}

func (repo *Repo) OpenRevision(id ID) (branch *Branch, err error) {
	return
}

func (repo *Repo) OpenBranch(src uint64) (branch *Branch, err error) {
	return
}

func (repo *Repo) Seal(branch *Branch) (err error) {
	return nil
}

func (repo *Repo) Close() (err error) {
	return nil
}
