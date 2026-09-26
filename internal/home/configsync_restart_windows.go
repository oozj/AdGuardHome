//go:build windows

package home

import (
	"os"
	"os/exec"

	"github.com/AdguardTeam/golibs/osutil"
)

func requestConfigSyncRestart() {
	executable, err := os.Executable()
	if err != nil {
		os.Exit(osutil.ExitCodeFailure)
	}

	err = exec.Command(executable, "--service", "restart").Start()
	if err != nil {
		os.Exit(osutil.ExitCodeFailure)
	}
}
