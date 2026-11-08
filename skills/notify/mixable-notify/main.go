// mixable-notify sends a Pushover notification to the user's phone with an
// "Open Rex" link. When run inside a Rex session, it also starts a background
// helper that jumps Rex for iOS to that session the next time the app
// reconnects (within 30 minutes; Rex for iOS needs Remote Control enabled).
//
//	go run <skill-dir>/mixable-notify/main.go "Refactored auth; tests pass."
//	echo "summary" | mixable-notify
//	mixable-notify -t "Custom title" "message"
//
// Pushover keys, first found wins: PUSHOVER_USER/PUSHOVER_TOKEN env vars,
// ~/.config/pushover/credentials, 1Password (op://Mixable/Pushover via the
// service account token in ~/.config/op/token), then the credentials file
// embedded at build time from this directory.
//
// Standard library only, so `go run main.go` works without a module.
package main

import (
	"bufio"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// The credentials file is optional (it's gitignored and only shipped in the
// claude.ai skill upload); "*" keeps the build working when it's absent.
//
//go:embed *
var bundled embed.FS

func bundledCreds() string {
	b, _ := bundled.ReadFile("credentials")
	return string(b)
}

const (
	pushoverAPI = "https://api.pushover.net/1/messages.json"
	openRexURL  = "shortcuts://run-shortcut?name=Open%20Rex"
	maxWait     = 30 * time.Minute
	maxMsgRunes = 1024
)

func main() {
	if len(os.Args) == 3 && os.Args[1] == "__jump" {
		os.Exit(jump(os.Args[2]))
	}

	title := flag.String("t", "", "title (default: <rex session label> · <host>)")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: mixable-notify [-t title] "summary"   (or pipe the summary on stdin)`)
	}
	flag.Parse()

	msg := strings.TrimSpace(strings.Join(flag.Args(), " "))
	if msg == "" && !isTerminal(os.Stdin) {
		b, _ := io.ReadAll(os.Stdin)
		msg = strings.TrimSpace(string(b))
	}
	if msg == "" {
		flag.Usage()
		os.Exit(2)
	}

	user, token := credentials()
	if user == "" || token == "" {
		fail("no Pushover keys (env, ~/.config/pushover/credentials, 1Password, or bundled)")
	}

	rex := rexBin()
	sid := os.Getenv("REX_SESSION")
	if *title == "" {
		label := ""
		if sid != "" && rex != "" {
			label = sessionLabel(rex, sid)
		}
		if label == "" {
			label = "agent"
		}
		*title = label + " · " + shortHost()
	}

	if err := send(user, token, *title, truncate(msg, maxMsgRunes)); err != nil {
		fail(err.Error())
	}
	fmt.Println("sent:", *title)

	if sid != "" && rex != "" {
		if err := startJump(sid); err != nil {
			fmt.Fprintln(os.Stderr, "mixable-notify: session jump not started:", err)
		}
	}
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, "mixable-notify:", msg)
	os.Exit(1)
}

// ---- credentials

func credentials() (user, token string) {
	if u, t := os.Getenv("PUSHOVER_USER"), os.Getenv("PUSHOVER_TOKEN"); u != "" && t != "" {
		return u, t
	}
	if b, err := os.ReadFile(filepath.Join(home(), ".config/pushover/credentials")); err == nil {
		if u, t := parseCreds(string(b)); u != "" && t != "" {
			return u, t
		}
	}
	if u, t := opPushover(); u != "" && t != "" {
		return u, t
	}
	return parseCreds(bundledCreds())
}

// parseCreds reads PUSHOVER_USER=... and PUSHOVER_TOKEN=... lines.
func parseCreds(s string) (user, token string) {
	for _, line := range strings.Split(s, "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		switch strings.TrimPrefix(strings.TrimSpace(k), "export ") {
		case "PUSHOVER_USER":
			user = v
		case "PUSHOVER_TOKEN":
			token = v
		}
	}
	return user, token
}

func opPushover() (user, token string) {
	op := findOp()
	tok, err := os.ReadFile(filepath.Join(home(), ".config/op/token"))
	if op == "" || err != nil || len(tok) == 0 {
		return "", ""
	}
	read := func(ref string) string {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, op, "read", ref)
		cmd.Env = append(withoutEnv(os.Environ(), "OP_SERVICE_ACCOUNT_TOKEN"),
			"OP_SERVICE_ACCOUNT_TOKEN="+strings.TrimSpace(string(tok)))
		out, err := cmd.Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(out))
	}
	return read("op://Mixable/Pushover/user key"), read("op://Mixable/Pushover/api token")
}

func findOp() string {
	if p, err := exec.LookPath("op"); err == nil {
		return p
	}
	for _, p := range []string{"/opt/homebrew/bin/op", "/usr/local/bin/op", filepath.Join(home(), ".local/bin/op")} {
		if isExecutable(p) {
			return p
		}
	}
	return ""
}

// ---- Pushover

func send(user, token, title, msg string) error {
	form := url.Values{
		"token":     {token},
		"user":      {user},
		"title":     {title},
		"message":   {msg},
		"url":       {openRexURL},
		"url_title": {"Open Rex"},
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.PostForm(pushoverAPI, form)
	if err != nil {
		return fmt.Errorf("Pushover request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	var r struct {
		Status int `json:"status"`
	}
	if json.Unmarshal(body, &r) != nil || r.Status != 1 {
		return fmt.Errorf("Pushover error (HTTP %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

// ---- Rex

func rexBin() string {
	if p, err := exec.LookPath("rex"); err == nil {
		return p
	}
	mac := filepath.Join(home(), "Library/Application Support/com.superlogical.rex/RexServer/Versions/Current/Rex Server.app/Contents/MacOS/rex")
	if isExecutable(mac) {
		return mac
	}
	return ""
}

func sessionLabel(rex, sid string) string {
	out, err := exec.Command(rex, "session", "inspect", sid, "--json").Output()
	if err != nil {
		return ""
	}
	var s struct {
		Label string `json:"label"`
	}
	_ = json.Unmarshal(out, &s)
	return s.Label
}

func stateDir() string { return filepath.Join(home(), ".config/pushover") }

// startJump replaces any running jump helper with one for this session.
func startJump(sid string) error {
	dir := stateDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if b, err := os.ReadFile(filepath.Join(dir, "jump.pid")); err == nil {
		if pid, err := strconv.Atoi(strings.TrimSpace(string(b))); err == nil && pid > 1 {
			_ = syscall.Kill(pid, syscall.SIGTERM)
		}
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	logf, err := os.OpenFile(filepath.Join(dir, "jump.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer logf.Close()
	cmd := exec.Command(exe, "__jump", sid)
	cmd.Stdout, cmd.Stderr = logf, logf
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

var errStop = errors.New("stop")

// jump waits for Rex for iOS to (re)connect, then selects the session.
func jump(sid string) int {
	logf := func(format string, a ...any) {
		fmt.Printf("%s "+format+"\n", append([]any{time.Now().Format("2006-01-02 15:04:05")}, a...)...)
	}
	rex := rexBin()
	if rex == "" {
		return 1
	}
	pidFile := filepath.Join(stateDir(), "jump.pid")
	_ = os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0o600)
	defer func() {
		if b, _ := os.ReadFile(pidFile); strings.TrimSpace(string(b)) == strconv.Itoa(os.Getpid()) {
			_ = os.Remove(pidFile)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), maxWait)
	defer cancel()

	var last time.Time
	try := func() error {
		if time.Since(last) < 3*time.Second {
			return nil
		}
		last = time.Now()
		time.Sleep(time.Second)
		for _, c := range iosClients(rex) {
			out, err := exec.Command(rex, "do", "-C", c, "session.select", "session_id="+sid).CombinedOutput()
			if err == nil {
				logf("jumped %s to %s", c, sid)
				return errStop
			}
			if strings.Contains(string(out), "not accepting actions") {
				logf("Rex for iOS refused: turn on Remote Control (Settings > Remote Hosts)")
				return errStop
			}
		}
		return nil
	}

	for ctx.Err() == nil {
		cmd := exec.CommandContext(ctx, rex, "events", "--all", "--json")
		stdout, err := cmd.StdoutPipe()
		if err == nil && cmd.Start() == nil {
			sc := bufio.NewScanner(stdout)
			sc.Buffer(make([]byte, 64<<10), 4<<20)
			for sc.Scan() {
				if !strings.Contains(sc.Text(), `"client_`) {
					continue
				}
				if try() == errStop {
					cancel()
					_ = cmd.Wait()
					return 0
				}
			}
			_ = cmd.Wait()
		}
		select { // stream ended (e.g. server restart); reconnect
		case <-ctx.Done():
		case <-time.After(5 * time.Second):
		}
	}
	return 0
}

func iosClients(rex string) []string {
	out, err := exec.Command(rex, "client", "ls", "--json").Output()
	if err != nil {
		return nil
	}
	var r struct {
		Clients []struct {
			ID    string `json:"client_id"`
			State string `json:"connection_state"`
			Info  struct {
				Platform string `json:"platform"`
			} `json:"info"`
		} `json:"clients"`
	}
	if json.Unmarshal(out, &r) != nil {
		return nil
	}
	var ids []string
	for _, c := range r.Clients {
		if c.Info.Platform == "ios" && c.State == "connected" {
			ids = append(ids, c.ID)
		}
	}
	return ids
}

// ---- helpers

func home() string {
	h, _ := os.UserHomeDir()
	return h
}

func shortHost() string {
	h, _ := os.Hostname()
	h, _, _ = strings.Cut(h, ".")
	return h
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func isExecutable(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir() && fi.Mode()&0o111 != 0
}

func withoutEnv(env []string, key string) []string {
	out := env[:0:0]
	for _, kv := range env {
		if !strings.HasPrefix(kv, key+"=") {
			out = append(out, kv)
		}
	}
	return out
}
