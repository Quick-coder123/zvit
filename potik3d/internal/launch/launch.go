// Package launch lets the game be updated by simply replacing potik.exe in
// its folder, even while someone else is playing.
//
// Windows locks a running .exe, so the game never runs from the shared
// folder: on start it copies itself into the user's local app-data folder
// (one copy per build, named by content hash) and relaunches from there.
// The file in the shared folder stays free to be overwritten; the next start
// picks up the new build, and a running game notices the change and offers
// a restart.
package launch

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	envShadow = "POTIK_SHADOW" // set in the relaunched copy
	envOrigin = "POTIK_ORIGIN" // path of the exe the player started
	prefix    = "potik-"
)

// Enabled reports whether shadow launching should be used on this system.
func Enabled() bool {
	return runtime.GOOS == "windows" || os.Getenv("POTIK_FORCE_SHADOW") == "1"
}

// Shadowed reports whether this process is the relaunched copy.
func Shadowed() bool { return os.Getenv(envShadow) == "1" }

// Origin is the path of the exe the player actually started.
func Origin() string {
	if p := os.Getenv(envOrigin); p != "" {
		return p
	}
	p, err := os.Executable()
	if err != nil {
		return ""
	}
	if r, err := filepath.EvalSymlinks(p); err == nil {
		p = r
	}
	return p
}

// CacheDir is where per-user copies of the game live.
func CacheDir() string {
	d, err := os.UserCacheDir()
	if err != nil {
		d = os.TempDir()
	}
	return filepath.Join(d, "Potik", "bin")
}

// Relaunch copies the running exe into CacheDir and starts the copy with the
// same arguments. It returns true when the caller should exit immediately.
// Any failure falls back to running in place.
func Relaunch() bool {
	if !Enabled() || Shadowed() {
		return false
	}
	origin := Origin()
	if origin == "" {
		return false
	}
	target, err := ShadowCopy(origin, CacheDir())
	if err != nil {
		return false
	}
	cmd := exec.Command(target, os.Args[1:]...)
	cmd.Env = append(cleanEnv(), envShadow+"=1", envOrigin+"="+origin)
	cmd.Dir = filepath.Dir(origin)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		return false
	}
	return true
}

func cleanEnv() []string {
	var out []string
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, envShadow+"=") || strings.HasPrefix(kv, envOrigin+"=") {
			continue
		}
		out = append(out, kv)
	}
	return out
}

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil))[:16], nil
}

// ShadowCopy makes sure dir holds a copy of exe named by its content hash and
// returns that path. Copies of other builds are removed when not in use.
func ShadowCopy(exe, dir string) (string, error) {
	sum, err := fileHash(exe)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	target := filepath.Join(dir, prefix+sum+filepath.Ext(exe))
	if _, err := os.Stat(target); err != nil {
		if err := copyFile(exe, target); err != nil {
			return "", err
		}
	}
	// best effort: copies still running for another session stay locked
	if entries, err := os.ReadDir(dir); err == nil {
		for _, e := range entries {
			p := filepath.Join(dir, e.Name())
			if strings.HasPrefix(e.Name(), prefix) && p != target {
				os.Remove(p)
			}
		}
	}
	return target, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + ".part"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}

// Watcher notices when the exe in the shared folder has been replaced.
type Watcher struct {
	path    string
	size    int64
	mod     time.Time
	next    time.Time
	changed bool
}

// NewWatcher starts watching path; nil if it cannot be read.
func NewWatcher(path string) *Watcher {
	st, err := os.Stat(path)
	if err != nil {
		return nil
	}
	return &Watcher{path: path, size: st.Size(), mod: st.ModTime()}
}

// Changed polls the file at most every few seconds.
func (w *Watcher) Changed() bool {
	if w == nil || w.changed {
		return w != nil && w.changed
	}
	now := time.Now()
	if now.Before(w.next) {
		return false
	}
	w.next = now.Add(10 * time.Second)
	st, err := os.Stat(w.path)
	if err != nil {
		return false // mid-copy or temporarily missing
	}
	// wait until the copy looks finished (unchanged for one poll)
	if st.Size() != w.size || !st.ModTime().Equal(w.mod) {
		if time.Since(st.ModTime()) > 3*time.Second {
			w.changed = true
		}
	}
	return w.changed
}

// Restart starts the exe from the shared folder (which relaunches the newest
// build). The caller should save and exit right after.
func Restart(origin string) error {
	cmd := exec.Command(origin, os.Args[1:]...)
	cmd.Env = cleanEnv()
	cmd.Dir = filepath.Dir(origin)
	return cmd.Start()
}
