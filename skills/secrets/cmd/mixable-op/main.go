// mixable-op runs the 1Password CLI as the "mixable-hosts" service account
// (read-only access to the Mixable vault), without touching how plain `op`
// behaves on the host.
//
//	mixable-op read "op://Mixable/<item>/<field>"
//	mixable-op item list --vault Mixable
package main

import (
	"fmt"
	"os"
	"syscall"

	"mixable-tools/internal/opsa"
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
	fmt.Fprintln(os.Stderr, "mixable-op:", err)
	os.Exit(1)
}
