//go:build !windows

package home

import (
	"os"
	"syscall"
)

func requestConfigSyncRestart() {
	configSyncRestartRequested.Store(true)
	_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
}
