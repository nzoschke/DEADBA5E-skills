# DEADBA5E-skills

Agent skills and Go tools for studio, mini, nuc and exe-rex (Go 1.27+, standard library only).
This repo is a Claude Code plugin marketplace with one plugin, `deadba5e`, whose skills load as `deadba5e:notify` and `deadba5e:secrets`.

| Path | What it is |
|---|---|
| `skills/notify` | Skill. `deadba5e-notify "summary"` pushes to the iPhone via Pushover with an Open Rex link and jumps Rex for iOS to the session. Standalone `main.go`, runs with `go run`. |
| `skills/secrets` | Skill + Go module. How agents use the DEADBA5E vault, plus the source for `deadba5e-op`, `deadba5e-askpass` (sudo -A), `deadba5e-sync` (SSH keys) and `deadba5e-claude` (Claude Code over SSH). |
| `.claude-plugin/` | Marketplace and plugin manifests. No `version`, so every pushed commit is an update. |

Install the plugin on a host (Claude Code):

    /plugin install deadba5e --marketplace nzoschke/DEADBA5E-skills

or from a shell (HTTPS, no GitHub SSH key needed):

    claude plugin marketplace add https://github.com/nzoschke/DEADBA5E-skills.git
    claude plugin install deadba5e@deadba5e

Then turn on auto-update in `/plugin` > Marketplaces > deadba5e > Enable auto-update, or run `claude plugin marketplace update deadba5e` after a push.

Install or update the Go tools on a host (from a clone at `~/DEADBA5E-skills`):

    git pull
    (cd skills/secrets && GOBIN=$HOME/.local/bin go install ./cmd/...)
    go build -o ~/.local/bin/deadba5e-notify skills/notify/deadba5e-notify/main.go
    for s in skills/*/; do ln -sfn "$PWD/$s" ~/.agents/skills/$(basename $s); done   # Codex and other agents

New host bootstrap (service account token) is described in `skills/secrets/SKILL.md`.

`skills/notify/deadba5e-notify/credentials` (Pushover keys) is gitignored, since this repo is public. Without it, `deadba5e-notify` reads the keys from 1Password.
