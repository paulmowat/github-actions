package plugin

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/drone-plugins/drone-github-actions/daemon"
	"github.com/drone-plugins/drone-github-actions/utils"
	"github.com/pkg/errors"
)

const (
	envFile          = "/tmp/action.env"
	secretFile       = "/tmp/action.secrets"
	workflowFile     = "/tmp/workflow.yml"
	eventPayloadFile = "/tmp/event.json"
)

var (
	secrets = []string{"GITHUB_TOKEN"}
)

type (
	Action struct {
		Uses         string
		With         map[string]string
		Env          map[string]string
		Image        string
		EventPayload string // Webhook event payload
		Actor        string
		Verbose      bool
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

	if err := utils.CreateEnvAndSecretFile(envFile, secretFile, secrets); err != nil {
		return err
	}

	cmdArgs := []string{
		"-W",
		workflowFile,
		"-P",
		fmt.Sprintf("ubuntu-latest=%s", p.Action.Image),
		"--secret-file",
		secretFile,
		"--env-file",
		envFile,
		"-b",
		"--detect-event",
	}

	// optional arguments
	if p.Action.Actor != "" {
		cmdArgs = append(cmdArgs, "--actor")
		cmdArgs = append(cmdArgs, p.Action.Actor)
	}

	if p.Action.EventPayload != "" {
		if err := ioutil.WriteFile(eventPayloadFile, []byte(p.Action.EventPayload), 0644); err != nil {
			return errors.Wrap(err, "failed to write event payload to file")
		}

		cmdArgs = append(cmdArgs, "--eventpath", eventPayloadFile)
	}

	if p.Action.Verbose {
		cmdArgs = append(cmdArgs, "-v")
	}

	var out bytes.Buffer
	cmd := exec.Command("act", cmdArgs...)
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return err
	}

	trace(&out)

	return nil
}

// trace writes each command to stdout with the command wrapped in an xml
// tag so that it can be extracted and displayed in the logs.
func trace(out *bytes.Buffer) {
	outputStr := out.String()
	fmt.Println(outputStr)

	lines := strings.Split(outputStr, "\n")

	// Prepare a map to hold the output values
	outputValues := make(map[string]string)

	// Iterate over lines to find ::set-output:: values
	for _, line := range lines {
		if strings.Contains(line, "::set-output::") {
			// Process line to extract set-output key and value
			parts := strings.Split(line, "::set-output::")
			if len(parts) > 1 {
				keyValue := strings.Split(parts[1], "=")
				if len(keyValue) > 1 {
					outputValues[keyValue[0]] = keyValue[1]
				}
			}
		}
	}

	// Prepare the output string in key=value format
	outputString := ""
	for key, value := range outputValues {
		outputString += fmt.Sprintf("%s=%s\n", key, value)
	}

	// Get the output file from the DRONE_OUTPUT environment variable
	outputFile := os.Getenv("DRONE_OUTPUT")

	// Write the output values to the output file
	err := os.WriteFile(outputFile, []byte(outputString), 0644)
	if err != nil {
		log.Fatal(err)
	}
}
