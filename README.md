# mixable-skills

Agent skills and Go tools for studio, mini, nuc and exe-rex (Go 1.27+, standard library only).
Both skills are uploaded to claude.ai (noah@housecat.com, personal org) and sync to every Claude Code login.

| Path | What it is |
|---|---|
| `skills/notify` | Skill. `mixable-notify "summary"` pushes to the iPhone via Pushover with an Open Rex link and jumps Rex for iOS to the session. Standalone `main.go`, runs with `go run`. |
| `skills/secrets` | Skill + Go module. How agents use the Mixable vault, plus the source for `mixable-op`, `mixable-askpass` (sudo -A), `mixable-sync` (SSH keys) and `mixable-claude` (Claude Code over SSH). |

Install or update on a host:

    (cd skills/secrets && GOBIN=$HOME/.local/bin go install ./cmd/...)
    go build -o ~/.local/bin/mixable-notify skills/notify/mixable-notify/main.go
    for s in skills/*/; do ln -sfn "$PWD/$s" ~/.agents/skills/$(basename $s); done   # Codex and other agents

New host bootstrap (service account token) is described in `skills/secrets/SKILL.md`.

Build the claude.ai uploads:

    (cd skills && for s in */; do rm -f ~/Downloads/${s%/}.zip; zip -qr ~/Downloads/${s%/}.zip ${s%/} -x '*.DS_Store'; done)

`skills/notify/mixable-notify/credentials` (Pushover keys) is gitignored. It's only in the claude.ai upload; without it, `mixable-notify` reads the keys from 1Password.
