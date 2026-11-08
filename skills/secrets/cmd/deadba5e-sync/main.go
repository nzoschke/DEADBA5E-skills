// deadba5e-sync writes every SSH key in the 1Password DEADBA5E vault to
// ~/.ssh/deadba5e/<name> (mode 600) and <name>.pub. Safe to re-run.
//
//	deadba5e-sync            # sync all SSH keys
//	ssh -i ~/.ssh/deadba5e/<name> host
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"deadba5e/internal/opsa"
)

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9._-]`)

func main() {
	h, _ := os.UserHomeDir()
	dir := flag.String("dir", filepath.Join(h, ".ssh/deadba5e"), "directory to write keys to")
	vault := flag.String("vault", "DEADBA5E", "1Password vault")
	flag.Parse()

	ctx := context.Background()
	out, err := opsa.Run(ctx, "item", "list", "--vault", *vault, "--categories", "SSH Key", "--format", "json")
	if err != nil {
		fail(err)
	}
	var items []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal(out, &items); err != nil {
		fail(fmt.Errorf("parse item list: %w", err))
	}
	if err := os.MkdirAll(*dir, 0o700); err != nil {
		fail(err)
	}

	n := 0
	for _, it := range items {
		name := unsafeName.ReplaceAllString(it.Title, "_")
		key, err := opsa.Run(ctx, "read", fmt.Sprintf("op://%s/%s/private key?ssh-format=openssh", *vault, it.ID))
		if err != nil {
			fmt.Fprintf(os.Stderr, "skip %s: %v\n", it.Title, err)
			continue
		}
		if err := writeKey(*dir, name, key); err != nil {
			fmt.Fprintf(os.Stderr, "skip %s: %v\n", it.Title, err)
			continue
		}
		fmt.Printf("synced %s -> %s\n", it.Title, filepath.Join(*dir, name))
		n++
	}
	fmt.Printf("%d key(s) synced\n", n)
}

// writeKey validates the key with ssh-keygen, then writes it and its public key atomically.
func writeKey(dir, name string, key []byte) error {
	tmp, err := os.CreateTemp(dir, ".tmp-")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0o600); err != nil {
		return err
	}
	if _, err := tmp.Write(key); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	pub, err := exec.Command("ssh-keygen", "-y", "-f", tmp.Name()).Output()
	if err != nil {
		return fmt.Errorf("private key unreadable by ssh-keygen: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".pub"), pub, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), filepath.Join(dir, name))
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "deadba5e-sync:", err)
	os.Exit(1)
}
