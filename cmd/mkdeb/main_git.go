package main

import (
	"github.com/chronos-tachyon/mkdeb"
)

var (
	Version    = "devel"
	Commit     = "devel"
	CommitDate = "devel"
	TreeState  = "devel"
)

func init() {
	mkdeb.SetVersion(
		"version",
		Version,
		"git.commit",
		Commit,
		"git.commitDate",
		CommitDate,
		"git.treeState",
		TreeState,
	)
}
