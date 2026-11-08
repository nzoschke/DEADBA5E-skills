# mixable-skills

Agent skills and Go tools for studio, mini, nuc and exe-rex (Go 1.27+, standard library only).
This repo is a Claude Code plugin marketplace with one plugin, `mixable`, whose skills load as `mixable:notify` and `mixable:secrets`.

| Path | What it is |
|---|---|
| `skills/notify` | Skill. `mixable-notify "summary"` pushes to the iPhone via Pushover with an Open Rex link and jumps Rex for iOS to the session. Standalone `main.go`, runs with `go run`. |
| `skills/secrets` | Skill + Go module. How agents use the Mixable vault, plus the source for `mixable-op`, `mixable-askpass` (sudo -A), `mixable-sync` (SSH keys) and `mixable-claude` (Claude Code over SSH). |
| `.claude-plugin/` | Marketplace and plugin manifests. No `version`, so every pushed commit is an update. |

Install the plugin on a host (Claude Code):

    /plugin install mixable --marketplace nzoschke/mixable-skills

or from a shell (HTTPS, no GitHub SSH key needed):

    claude plugin marketplace add https://github.com/nzoschke/mixable-skills.git
    claude plugin install mixable@mixable

Then turn on auto-update in `/plugin` > Marketplaces > mixable > Enable auto-update, or run `claude plugin marketplace update mixable` after a push.

Install or update the Go tools on a host (from a clone at `~/mixable-skills`):

    git pull
    (cd skills/secrets && GOBIN=$HOME/.local/bin go install ./cmd/...)
    go build -o ~/.local/bin/mixable-notify skills/notify/mixable-notify/main.go
    for s in skills/*/; do ln -sfn "$PWD/$s" ~/.agents/skills/$(basename $s); done   # Codex and other agents

New host bootstrap (service account token) is described in `skills/secrets/SKILL.md`.

`skills/notify/mixable-notify/credentials` (Pushover keys) is gitignored, since this repo is public. Without it, `mixable-notify` reads the keys from 1Password.
