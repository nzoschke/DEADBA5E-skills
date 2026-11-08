// deadba5e-op runs the 1Password CLI as the "DEADBA5E" service account
// (read-only access to the DEADBA5E vault), without touching how plain `op`
// behaves on the host.
//
//	deadba5e-op read "op://DEADBA5E/<item>/<field>"
//	deadba5e-op item list --vault DEADBA5E
package main

import (
	"fmt"
	"os"
	"syscall"

	"deadba5e/internal/opsa"
)

func main() {
	op, err := opsa.Op()
	if err != nil {
		fail(err)
	}
	env, err := opsa.Env()
	if err != nil {
		fail(err)
	}
	fail(syscall.Exec(op, append([]string{op}, os.Args[1:]...), env))
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "deadba5e-op:", err)
	os.Exit(1)
}
