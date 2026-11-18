---
name: nuc
description: Reboot the nuc (ASUS NUC 15 Pro) between Ubuntu (`nuc`) and Windows 11 (`nuc-win`) over SSH with boot-to, tell which OS it's running, and recover when it doesn't come back. Use whenever a task needs Linux or Windows on the nuc, the user asks to reboot it into the other OS, or nuc / nuc-win isn't answering.
---

# nuc

The nuc dual boots, so it's either `nuc` (Ubuntu 26.04) or `nuc-win` (Windows 11 Pro), never both. Each OS is its own Tailscale node with its own SSH, and they share one LAN MAC (DHCP, currently 10.0.0.65). Switching takes about 35 s from Linux to Windows and about 50 s back. Nobody has to be at the machine.

## Which OS is it in?

```bash
tailscale status --json | jq -r '.Peer[] | select(.HostName=="nuc" or .HostName=="nuc-win") | "\(.HostName) \(.Online)"'
nc -z -G 3 nuc 22 && echo linux; nc -z -G 3 nuc-win 22 && echo windows
```

Use the port check. Tailscale's Online flag can lag a minute or two after a switch.

## Switch OS

```bash
ssh nuc boot-to windows          # from Linux; then wait for nuc-win
ssh nuc-win boot-to linux        # from Windows; then wait for nuc
```

`--no-reboot` (Linux) or `-NoReboot` (Windows) only sets the next boot.

Then wait for the other side's SSH port, without a fixed sleep:

```bash
until nc -z -G 2 nuc-win 22 2>/dev/null; do sleep 5; done; ssh -o BatchMode=yes nuc-win hostname
```

Run the wait in the background with a time limit of about 5 minutes. If it runs out, see "Recovery".

From the mini, `ssh nuc` and `ssh nuc-win` work without 1Password (see the `ssh` skill). From any other host, set up a key first, or the call will hang on a 1Password prompt.

## How it works

- `boot-to` sets the UEFI **BootNext** variable, a one-time choice, and reboots. **Ubuntu stays first in BootOrder**, so a crash, a power cut or a Windows Update restart always lands in Linux. Linux is the home base.
- Linux: `/usr/local/bin/boot-to windows|linux [--no-reboot]` uses `efibootmgr --bootnext` through sudo. It finds `deadba5e-askpass` itself, so it works over plain SSH.
- Windows: `C:\Windows\boot-to.cmd` → `C:\ProgramData\boot-to.ps1` uses `bcdedit /set {fwbootmgr} bootsequence <Ubuntu entry>`. It finds the Ubuntu entry by `\EFI\ubuntu\shimx64.efi` and puts Ubuntu back first in the display order every time, which undoes Windows Update moving itself first. SSH sessions as `noah` (admin) are already elevated.
- Windows signs `noah` in at boot by itself, never sleeps, has hibernation and Fast Startup off, keeps the clock in UTC (`RealTimeIsUniversal=1`), and doesn't restart for updates on its own. SSH lands in PowerShell.

## Running work on the other side

The loop for an agent:
1. Check which OS is up.
2. Run `boot-to <other>` and wait for the port.
3. Do the work over SSH.
4. If the user wants Linux left running, switch back.

Say in your reply which OS you left it in. The nuc's Rex server (`nuc-rex`) runs only on Linux; Windows is reachable by SSH only.

## Gotchas

- `efibootmgr --delete-bootnext` is buggy on this firmware. To clear a pending BootNext on Linux, remove the variable directly:
  ```bash
  sudo chattr -i /sys/firmware/efi/efivars/BootNext-8be4df61-93ca-11d2-aa0d-00e098032b8c; sudo rm /sys/firmware/efi/efivars/BootNext-8be4df61-93ca-11d2-aa0d-00e098032b8c
  ```
- The firmware adds a "UEFI OS" boot entry by itself for any FAT partition with `\EFI\BOOT\BOOTX64.EFI`. Check `sudo efibootmgr -v` after changing partitions.
- BIOS settings can't be changed remotely: no vPro/AMT, and `asus-armoury` exposes nothing. To get into BIOS setup, run `ssh nuc sudo systemctl reboot --firmware-setup`, and someone has to be at the TV and keyboard. The user's Apple keyboard doesn't work in the BIOS.
- Don't run root commands on the nuc's partitions or boot entries beyond `boot-to` without the user's go-ahead. The disk layout is p1 1 GB ESP (shared), p2 1.45 TB Ubuntu, p3 MSR, p4 ~400 GB Windows, p5 WinRE, and ~12 GB unallocated.

## Recovery

- **Back in Linux when you expected Windows:** BootNext was consumed or ignored. Check `sudo efibootmgr` for the Windows Boot Manager entry and retry once.
- **Neither side answers after about 5 minutes:** it's stuck on a screen, or it's in Windows without network. Ask the user to look at the TV. If it's in Windows, they can open an admin terminal (Win+X, A, then Left arrow and Enter to approve) and run `boot-to linux`. If that fails, run `shutdown /r /o /t 0`, then pick **Use a device → Ubuntu**. A hard power-off (hold the button for 5 s) also lands in Linux.
- **Windows is up but not on Tailscale:** usually a missing network driver after a big update. Boot Linux and tell the user.
