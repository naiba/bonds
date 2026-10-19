package dav

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func openCardDAVPhoto(root, name string) (*os.File, error) {
	// Reading a known UUID needs directory search permission, not permission to
	// list uploads. os.OpenRoot uses O_RDONLY and breaks mode-0300 upload roots.
	// O_PATH plus RESOLVE_BENEATH preserves that permission model while the
	// kernel confines path traversal and symlinks, including concurrent renames.
	directory, err := unix.Open(root, unix.O_PATH|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	defer unix.Close(directory)
	fd, err := unix.Openat2(directory, name, &unix.OpenHow{
		Flags:   unix.O_RDONLY | unix.O_CLOEXEC,
		Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_MAGICLINKS,
	})
	if errors.Is(err, unix.ENOSYS) || errors.Is(err, unix.EPERM) {
		// Older kernels or syscall policies may not provide openat2. Keep the
		// standard library's confinement; never fall back to an unchecked open.
		return os.OpenInRoot(root, name)
	}
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), name), nil
}
