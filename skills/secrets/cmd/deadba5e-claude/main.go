// deadba5e-claude runs Claude Code with the long-lived subscription token from
// the 1Password item "Claude Code token" in the DEADBA5E vault, so it works over
// SSH on Macs where the Keychain login is unavailable.
//
//	ssh mini deadba5e-claude -p "Reply with just: ok"
//
// If CLAUDE_CODE_OAUTH_TOKEN is already set, it is used as is. If the item
// can't be read, Claude Code runs with its normal login.
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"deadba5e/internal/opsa"
)

const (
	tokenEnv = "CLAUDE_CODE_OAUTH_TOKEN"
	tokenRef = "op://DEADBA5E/Claude Code token/credential"
)

func main() {
	claude, err := findClaude()
	if err != nil {
		fail(err)
	}
	env := os.Environ()
	if os.Getenv(tokenEnv) == "" {
		if tok, err := opsa.Run(context.Background(), "read", tokenRef); err == nil {
			if t := strings.TrimSpace(string(tok)); t != "" {
				env = append(env, tokenEnv+"="+t)
			}
		} else {
			fmt.Fprintln(os.Stderr, "deadba5e-claude: no token from 1Password, using normal login:", err)
		}
	}
	fail(syscall.Exec(claude, append([]string{claude}, os.Args[1:]...), env))
}

func findClaude() (string, error) {
	if p, err := exec.LookPath("claude"); err == nil {
		return p, nil
	}
	h, _ := os.UserHomeDir()
	for _, p := range []string{filepath.Join(h, ".local/bin/claude"), "/usr/local/bin/claude", "/opt/homebrew/bin/claude"} {
		if fi, err := os.Stat(p); err == nil && fi.Mode()&0o111 != 0 {
			return p, nil
		}
	}
	return "", fmt.Errorf("claude not found")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "deadba5e-claude:", err)
	os.Exit(1)
}
