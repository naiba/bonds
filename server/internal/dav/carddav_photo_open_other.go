//go:build !linux

package dav

import "os"

func openCardDAVPhoto(root, name string) (*os.File, error) {
	return os.OpenInRoot(root, name)
}
