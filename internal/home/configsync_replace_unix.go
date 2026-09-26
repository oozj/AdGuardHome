//go:build !windows

package home

import "os"

func replaceSyncedConfig(from, to string) (err error) {
	return os.Rename(from, to)
}
