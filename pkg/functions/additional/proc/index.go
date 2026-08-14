// Package proc exposes process and user context (uid, gid, cwd)
package proc

import (
	"os"
	"os/user"
	"text/template"
)

func FuncMap() template.FuncMap {
	return template.FuncMap{
		"uid":     os.Getuid,
		"gid":     os.Getgid,
		"cwd":     os.Getwd,
	}
}

func currentUser() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return u.Username, nil
}
