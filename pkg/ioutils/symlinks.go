package ioutils

import (
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/sftp"
)

func ResolveSymlinkIfNeededLocalfs(p string) (string, error) {
	fi, err := os.Lstat(p)
	if err != nil {
		return "", err
	}

	if fi.Mode()&os.ModeSymlink != 0 {
		// It's a symlink — resolve it
		resolvedPath, err := filepath.EvalSymlinks(p)
		if err != nil {
			return "", err
		}
		return resolvedPath, nil
	}

	return p, nil
}

func ResolveSymlinkIfNeededSftp(remotePath string, sftpClient *sftp.Client) (string, error) {
	fi, err := sftpClient.Lstat(remotePath)
	if err != nil {
		return "", err
	}

	if fi.Mode()&os.ModeSymlink == 0 {
		return remotePath, nil // not a symlink
	}

	target, err := sftpClient.ReadLink(remotePath)
	if err != nil {
		return "", err
	}

	// If the symlink is relative, resolve it relative to the dir of the symlink
	if !path.IsAbs(target) {
		target = path.Join(path.Dir(remotePath), target)
	}

	// Recurse once — in a case of nested symlinks
	return ResolveSymlinkIfNeededSftp(target, sftpClient)
}

func CreateSymlink(target, link string, force bool) error {
	if force {
		// Remove existing link or file
		if _, err := os.Lstat(link); err == nil {
			err = os.Remove(link)
			if err != nil {
				return err
			}
		}
	}

	// Create the symlink
	return os.Symlink(target, link)
}
