package util

import (
	"os"
	"path/filepath"
)

const (
	BunnyshellWorkspaceDirname = ".bunnyshell"
	RemoteDevDirname           = "remote-dev"
)

func GetWorkspaceDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	if home == "/" {
		return "/bunnyshell", nil
	}

	return filepath.Join(home, BunnyshellWorkspaceDirname), nil
}

func GetRemoteDevWorkspaceDir() (string, error) {
	workspaceDir, err := GetWorkspaceDir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(workspaceDir, RemoteDevDirname)
	os.MkdirAll(path, 0700)

	return path, nil
}
