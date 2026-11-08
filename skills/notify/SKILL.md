---
name: notify
description: Send a push notification to the user's iPhone with a short summary of what you did, plus a link that opens Rex at this session. Use when the user asks to be notified, pinged, buzzed or pushed ("notify me when done", "ping my phone", "/notify"), or when they told you earlier in the task that they'd be away and you've finished or are blocked on them.
---

# notify

Send one push with the Go program bundled in this skill. If `deadba5e-notify` is on PATH, run that; otherwise `go run` the bundled source using its full path under this skill's directory (the directory this SKILL.md is in):

```bash
deadba5e-notify "Migrated the users table and updated the 3 queries that used it. Tests pass; nothing needs you."
# or
go run <skill-dir>/deadba5e-notify/main.go "Migrated the users table and updated the 3 queries that used it. Tests pass; nothing needs you."
```

It needs only the Go toolchain (standard library, no module needed). Pushover keys come from env vars, `~/.config/pushover/credentials`, 1Password (`op://DEADBA5E/Pushover`), or the copy embedded from `deadba5e-notify/credentials`, in that order. Inside a Rex session it also makes Rex for iOS jump to this session when the user taps the notification.

When to send:
- **Finished the task:** send it as your last step, after the work is verified.
- **Blocked on the user:** a question, a decision, or access you need. Send it right before you stop, and say what you need.
- Send one push per stopping point. Don't send progress updates unless the user asked for them.

Writing the summary:
- One or two plain sentences, under about 200 characters. It's read on a lock screen.
- Say what you did and what the result was, then whether the user needs to act ("nothing needs you", "needs your call on X").
- No markdown, code blocks or file dumps. Name at most one or two files if it helps.
- The title is set automatically to `<rex session label> · <host>`. Pass `-t "Title"` only if the user asked for a specific title.

If the command fails (it prints `deadba5e-notify: ...`), tell the user why in your reply. Don't retry in a loop.
