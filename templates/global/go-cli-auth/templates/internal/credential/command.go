package credential

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func runCommand(ctx context.Context, command, configPath string) (Credential, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return Credential{}, fmt.Errorf("run the credentialCommand in %s: %w", configPath, err)
	}
	value := strings.TrimSpace(string(out))
	if value == "" {
		return Credential{}, fmt.Errorf("the credentialCommand in %s printed no {{ .Computed.label }}", configPath)
	}
	return Credential{Value: value, Source: "the credentialCommand in " + configPath}, nil
}
