// deadba5e-askpass is a SUDO_ASKPASS helper: it prints this host's sudo
// password from the 1Password item "sudo <short hostname>" in the DEADBA5E
// vault, so agents can run `sudo -A <cmd>` unattended.
//
//	export SUDO_ASKPASS=$HOME/.local/bin/deadba5e-askpass
//	sudo -A apt install jq
//
// Override the item with DEADBA5E_SUDO_ITEM (title or ID).
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"deadba5e/internal/opsa"
)

func main() {
	item := os.Getenv("DEADBA5E_SUDO_ITEM")
	if item == "" {
		h, _ := os.Hostname()
		h, _, _ = strings.Cut(h, ".")
		item = "sudo " + h
	}
	pw, err := opsa.Run(context.Background(), "read", "op://DEADBA5E/"+item+"/password")
	if err != nil {
		fmt.Fprintf(os.Stderr, "deadba5e-askpass: %v\n", err)
		os.Exit(1)
	}
	pw = bytes.TrimRight(pw, "\r\n")
	if len(pw) == 0 {
		fmt.Fprintf(os.Stderr, "deadba5e-askpass: %q has an empty password\n", item)
		os.Exit(1)
	}
	os.Stdout.Write(append(pw, '\n'))
}
