package runtime

import "os/exec"

func KillProcessTree(cmd *exec.Cmd) {
	killTree(cmd)
}
