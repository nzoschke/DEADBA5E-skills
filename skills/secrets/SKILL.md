---
name: secrets
description: Get secrets, SSH keys and root on the user's machines (studio, mini, nuc, exe-rex) from the 1Password "DEADBA5E" vault, and run Claude Code on another host over SSH. Use whenever a task needs an API key, password, token or other credential, an SSH key for another host, sudo/root (installing packages, editing system files, managing services), or a Claude Code run on a different machine — instead of asking the user to paste a secret or type a password. Also covers installing these tools on a new host.
---

# secrets

Every host has read-only access to the 1Password **DEADBA5E** vault through a service account. These tools are installed in `~/.local/bin` (use the full path if it isn't on PATH). Their Go source is bundled in this skill (see "Set up a host").

| Tool | Use |
|---|---|
| `deadba5e-op` | The 1Password CLI, signed in as the service account. |
| `deadba5e-askpass` | Supplies this host's sudo password to `sudo -A`. |
| `deadba5e-sync` | Writes every SSH key in the vault to `~/.ssh/deadba5e/<name>`. |
| `deadba5e-claude` | Runs `claude` with the long-lived token from the vault, so Claude Code works over SSH (including on the Macs). |

## Find and read a secret

```bash
deadba5e-op item list --vault DEADBA5E                        # what's available
deadba5e-op item get "<item>" --vault DEADBA5E --format json | jq -c '[.fields[] | {label, type}]'  # field names only (raw JSON includes values)
deadba5e-op read "op://DEADBA5E/<item>/<field>"               # one value
```

Use the value without printing it:
- Pass it inline to the command that needs it: `API_KEY=$(deadba5e-op read "op://DEADBA5E/Foo/credential") ./deploy`, or pipe it on stdin.
- To run a command with several secrets as env vars: `deadba5e-op run --env-file=.env.tpl -- <cmd>`, where `.env.tpl` holds lines like `FOO=op://DEADBA5E/Foo/credential`.
- Never echo a secret, put it in your reply, log it, or write it into a repo, a commit or a file other tools read. If a value must go into a config file, tell the user which file and keep that file out of version control.

Use `deadba5e-op`, not plain `op`: on the Macs, plain `op` uses the user's personal sign-in and may prompt for Touch ID.

## sudo / root

```bash
SUDO_ASKPASS=~/.local/bin/deadba5e-askpass sudo -A <command>
```

This reads the password from the item `sudo <short hostname>` (for example `sudo mini`). exe-rex has passwordless sudo, so plain `sudo` works there too. Root on these hosts is allowed; still say in your reply what you ran as root.

## SSH keys

```bash
deadba5e-sync                                      # refresh ~/.ssh/deadba5e/ from the vault
ssh -i ~/.ssh/deadba5e/<name> -o IdentitiesOnly=yes <host>
```

The `mixable` key is the noah@mixable.net exe.dev account (VMs like `exe-rex.exe.xyz`).

## Claude Code on another host

```bash
ssh <host> '~/.local/bin/deadba5e-claude -p "<task>"'
```

Use `deadba5e-claude`, not plain `claude`, over SSH: on the Macs plain `claude` can't reach its Keychain login and reports "Not logged in". These sessions get the `deadba5e` plugin skills but not skills uploaded to claude.ai.

## Set up a host

The Go source for all of these tools is in this skill directory (`go.mod`, `cmd/`, `internal/`; Go 1.27+, standard library only). To install or update them on the current host:

```bash
cd <skill-dir> && GOBIN=$HOME/.local/bin go install ./cmd/...
```

A brand-new host also needs the service account token, which only the user can provide (bootstrap): it's the item "DEADBA5E service account" in DEADBA5E, and it goes in `~/.config/op/token` (mode 600). Pipe it from a host where the user is signed in to 1Password, never through a command line or a chat:

```bash
op read "op://DEADBA5E/DEADBA5E service account/credential" | ssh <new-host> 'umask 077; mkdir -p ~/.config/op; cat > ~/.config/op/token'
```

Then on the new host: add a `sudo <short hostname>` item to DEADBA5E (the user does this), run `deadba5e-sync`, and install these skills as a Claude Code plugin:

```bash
claude plugin marketplace add https://github.com/nzoschke/DEADBA5E-skills.git
claude plugin install deadba5e@deadba5e
```

## Limits

- **Read-only.** You can't add or change items. If a secret is missing, tell the user the item and field name to create in DEADBA5E.
- If `~/.config/op/token` is missing, the user has to set the host up; don't try other ways in.
