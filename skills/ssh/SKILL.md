---
name: ssh
description: Find which of the user's machines (studio, mini, nuc, nuc-win, work, exe-rex) are online and reachable over SSH, and set up host-to-host SSH with keys from the 1Password DEADBA5E vault so agents connect without Touch ID or 1Password approval prompts. Use when a task needs to reach another host, when SSH fails with "agent refused operation", "communication with agent failed" or a 1Password prompt nobody can approve, or when setting up a new host-to-host connection.
---

# ssh

## Hosts

All hosts are on one Tailscale tailnet (MagicDNS names below), and most are also on the home LAN (10.0.0.0/24).

| Host | OS | How to reach it | Notes |
|---|---|---|---|
| `mini` | macOS | `mini` | Home Mac. Accepts the `deadba5e` key. |
| `studio` | macOS laptop | `studio`, or `studio.local` on the home LAN | Accepts `deadba5e`. If `ssh studio` times out, Tailscale on the studio is probably stuck: check `tailscale ping studio`, then use `studio.local`. |
| `work` | macOS | `work` | Same: port 22 times out. |
| `nuc` | Ubuntu 26.04 | `nuc` | Same machine as `nuc-win`; only one is up at a time. See the `nuc` skill. |
| `nuc-win` | Windows 11 Pro | `nuc-win` | Logs in to PowerShell as `noah` (local admin). |
| `exe-rex` | Linux VM on exe.dev | Rex (`rex -S exe …`) | No agent SSH. exe.dev's SSH proxy (`*.exe.xyz`) checks keys on the exe.dev *account*, not the VM's `authorized_keys`, and the VM runs no sshd on its tailnet IP. |

`*-rex` tailnet nodes (`mini-rex`, `nuc-rex`, …) are Rex servers, not SSH hosts.

## Find what's online

```bash
# Tailscale: name, OS, online. Fastest and covers every host.
tailscale status --json | jq -r '.Peer[] | select(.HostName != "funnel-ingress-node") | [.HostName, .OS, (if .Online then "online" else "offline" end), .TailscaleIPs[0]] | @tsv'

# Is sshd answering? (macOS nc; on Linux use `nc -z -w 3`)
for h in studio mini work nuc nuc-win; do nc -z -G 3 $h 22 2>/dev/null && echo "$h ssh up" || echo "$h ssh down"; done

# LAN sweep for anything running sshd, with its banner (shows Windows vs Linux vs embedded devices)
for i in $(seq 1 254); do (nc -z -G 1 10.0.0.$i 22 2>/dev/null && echo "10.0.0.$i $(ssh-keyscan -T 2 10.0.0.$i 2>/dev/null | head -1 | cut -d' ' -f3-)") & done; wait

# Macs advertising Remote Login over Bonjour (the local Mac always shows up)
dns-sd -B _ssh._tcp local.     # Ctrl-C after a few seconds
```

Then prove you can log in without prompts. `BatchMode=yes` fails immediately instead of waiting on a password or 1Password:

```bash
ssh -o BatchMode=yes -o ConnectTimeout=5 <host> hostname
```

"Online in Tailscale" but "ssh down" usually means Remote Login (macOS) or sshd is off on that host. Tell the user; don't try to turn it on remotely.

## Never trigger a 1Password prompt

On the Macs, `~/.ssh/config` has `Host *` → `IdentityAgent` = the 1Password agent. Any `ssh` that falls through to it asks for Touch ID or approval, which nobody may be there to give. It fails with `agent refused operation` or `communication with agent failed`, or it just hangs. Passing `-i <key>` is **not** enough: ssh still asks the agent first.

Always use a key file with the agent turned off:

```bash
ssh -o IdentityAgent=none -o IdentitiesOnly=yes -i ~/.ssh/deadba5e/<key> <host>
```

Or, better, a config block for the host (see below) so a plain `ssh <host>` works for every tool. If the only way in is the 1Password agent, stop and ask the user, saying they'll get an approval prompt. Don't retry in a loop, since each try pops a new prompt.

## SSH keys in 1Password

SSH keys live as **SSH Key** items in the DEADBA5E vault. Every host can read them through the `secrets` skill's service account. No Touch ID is involved.

```bash
deadba5e-op item list --vault DEADBA5E --categories "SSH Key"   # which keys exist
deadba5e-sync                                                     # write all of them to ~/.ssh/deadba5e/<name> (+ .pub)
```

**Use the `deadba5e` key for host-to-host SSH.** It exists only for these machines. Don't put keys that open other systems in this vault, such as the exe.dev account key: every host syncs every key in it.

Agents can read the vault but can't add to it. If a key is missing, ask the user to create one: in 1Password, New Item → **SSH Key** → Generate (Ed25519), in vault **DEADBA5E**, named for what it opens (for example `nuc-agent`).

## Set up approval-free SSH from host A to host B

1. **Get the key onto A.** On A: `deadba5e-sync`, then check `~/.ssh/deadba5e/<key>` exists.
2. **Authorize the key on B.** Append `~/.ssh/deadba5e/<key>.pub` to B's authorized keys. This needs some way into B that already works: an existing key, Rex (`rex -S <host> new …` then `run --wait`; see the `rex` skill, and kill the session afterwards), or one 1Password approval the user gives while they're present. Adding a login key is persistent access, so get the user's explicit go-ahead for each host first.
   - **Linux/macOS:**
     ```bash
     ssh B 'umask 077; mkdir -p ~/.ssh; cat >> ~/.ssh/authorized_keys' < ~/.ssh/deadba5e/<key>.pub
     ```
   - **Windows** (admin users read only the ProgramData file, and sshd ignores it if its permissions are loose):
     ```bash
     K="$(cut -d' ' -f1,2 ~/.ssh/deadba5e/<key>.pub) <key>"
     PS='$k = "'"$K"'"
     $f = "C:\ProgramData\ssh\administrators_authorized_keys"
     if ((Get-Content $f) -match [regex]::Escape($k.Split(" ")[1])) { "already" } else {
       Add-Content -Path $f -Value $k
       icacls $f /inheritance:r /grant Administrators:F /grant SYSTEM:F | Out-Null
       "added" }'
     ssh B "powershell -NoProfile -EncodedCommand $(printf '%s' "$PS" | iconv -t UTF-16LE | base64)"
     ```
     This skips a key that's already there. On Linux/macOS, check first with `grep -F "$(cut -d' ' -f2 <key>.pub)" ~/.ssh/authorized_keys` on B.
3. **Add a config block on A**, *above* `Host *`, so plain `ssh B` uses the file key and never the 1Password agent. Back up `~/.ssh/config` first:
   ```
   Host nuc nuc-win
   	IdentityAgent none
   	IdentityFile ~/.ssh/deadba5e/deadba5e
   	IdentitiesOnly yes
   	ControlMaster auto
   	ControlPath ~/.ssh/cm/%r@%h:%p
   	ControlPersist 4h
   ```
   Run `mkdir -p ~/.ssh/cm` for the ControlPath. The quoted 1Password socket path in `Host *` must stay quoted, because it contains spaces.
4. **Test** from A, ignoring the config so another key can't make it pass by accident (`IdentityFile` lines add up):
   ```bash
   ssh -F /dev/null -o BatchMode=yes -o IdentityAgent=none -o IdentitiesOnly=yes -o UserKnownHostsFile=$HOME/.ssh/known_hosts -i ~/.ssh/deadba5e/deadba5e B hostname
   ```
   On Windows, send PowerShell as `powershell -NoProfile -EncodedCommand <base64 UTF-16LE>` (`printf '%s' "$ps" | iconv -t UTF-16LE | base64`). SSH lands in PowerShell there, and nested quoting falls apart. Accept a new host key with `-o StrictHostKeyChecking=accept-new`, but only for the user's own tailnet hosts.

Report which hosts you authorized and which key you used.

## Current state (2026-11-18)

- Vault SSH keys: only `deadba5e`.
- `deadba5e` is authorized on **mini**, **studio**, **nuc** and **nuc-win**. It is not on **work** (sshd unreachable). exe-rex is Rex-only.
- `~/.ssh/config` on **mini**, **studio** and **nuc** has a `# DEADBA5E hosts` block, so plain `ssh <host>` between them (and to nuc-win) skips 1Password. To add it on another host, put the step 3 block above `Host *`, never above an `Include` line, or it captures the Include.
