// Package opsa runs the 1Password CLI as the "DEADBA5E" service account,
// whose token lives in ~/.config/op/token (read-only access to the DEADBA5E vault).
package opsa

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const tokenEnv = "OP_SERVICE_ACCOUNT_TOKEN"

// Op returns the path to the 1Password CLI.
func Op() (string, error) {
	if p, err := exec.LookPath("op"); err == nil {
		return p, nil
	}
	h, _ := os.UserHomeDir()
	for _, p := range []string{"/opt/homebrew/bin/op", "/usr/local/bin/op", filepath.Join(h, ".local/bin/op")} {
		if fi, err := os.Stat(p); err == nil && fi.Mode()&0o111 != 0 {
			return p, nil
		}
	}
	return "", errors.New("1Password CLI (op) not found")
}

// TokenPath is where the service account token is stored.
func TokenPath() string {
	h, _ := os.UserHomeDir()
	return filepath.Join(h, ".config/op/token")
}

// Env returns os.Environ() with the service account token set.
func Env() ([]string, error) {
	b, err := os.ReadFile(TokenPath())
	if err != nil || len(bytes.TrimSpace(b)) == 0 {
		return nil, fmt.Errorf("missing service account token in %s", TokenPath())
	}
	env := make([]string, 0, len(os.Environ())+1)
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, tokenEnv+"=") {
			env = append(env, kv)
		}
	}
	return append(env, tokenEnv+"="+string(bytes.TrimSpace(b))), nil
}

// Run runs op with args as the service account and returns stdout.
func Run(ctx context.Context, args ...string) ([]byte, error) {
	op, err := Op()
	if err != nil {
		return nil, err
	}
	env, err := Env()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, op, args...)
	cmd.Env = env
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("op %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}
