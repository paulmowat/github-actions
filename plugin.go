package plugin

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/drone-plugins/drone-github-actions/daemon"
	"github.com/drone-plugins/drone-github-actions/utils"
)

const (
	workflowFile = "/tmp/workflow.yml"
	webhookFile  = "/tmp/webhook"
	envFile      = "/tmp/action.env"
)

type (
	Action struct {
		Uses    string
		With    map[string]string
		Env     map[string]string
		Image   string
		Verbose bool
	}

	Plugin struct {
		Action Action
		Daemon daemon.Daemon // Docker daemon configuration
	}
)

// Exec executes the plugin step
func (p Plugin) Exec() error {
	if err := daemon.StartDaemon(p.Daemon); err != nil {
		return err
	}

	if err := utils.CreateWorkflowFile(workflowFile, p.Action.Uses,
		p.Action.With, p.Action.Env); err != nil {
		return err
	}

	cmdArgs := []string{
		"-W",
		workflowFile,
		"-P",
		fmt.Sprintf("ubuntu-latest=%s", p.Action.Image),
		"--detect-event",
	}

	if p.Action.Verbose {
		cmdArgs = append(cmdArgs, "-v")
	}

	cmd := exec.Command("act", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	trace(cmd)

	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

// trace writes each command to stdout with the command wrapped in an xml
// tag so that it can be extracted and displayed in the logs.
func trace(cmd *exec.Cmd) {
	fmt.Fprintf(os.Stdout, "+ %s\n", strings.Join(cmd.Args, " "))
}