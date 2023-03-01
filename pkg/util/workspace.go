package util

import (
	"os"
	"path/filepath"
)

const (
	BunnyshellWorkspaceDirname = ".bunnyshell"
	RemoteDevDirname           = "remote-dev"

	dirPermissionMask = 0700
)

var remoteDevWorkspace *string

func GetRemoteDevWorkspaceDir() (string, error) {
	if remoteDevWorkspace != nil {
		return *remoteDevWorkspace, nil
	}

	path, err := ensureRemoteDevWorkspaceDir()
	if err != nil {
		return "", nil
	}

	remoteDevWorkspace = &path

	return path, nil
}

func getWorkspaceDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	if home == "/" {
		return "/bunnyshell", nil
	}

	return filepath.Join(home, BunnyshellWorkspaceDirname), nil
}

func ensureRemoteDevWorkspaceDir() (string, error) {
	workspaceDir, err := getWorkspaceDir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(workspaceDir, RemoteDevDirname)
	if err = os.MkdirAll(path, dirPermissionMask); err != nil {
		return "", err
	}

	return path, nil
}
