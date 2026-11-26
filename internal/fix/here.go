package fix

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Fd3os/pathdoc/internal/translit"
	"github.com/Fd3os/pathdoc/internal/win"
)

// ErrAlreadyASCII: the project needs no junction.
var ErrAlreadyASCII = errors.New("the project path is plain ASCII")

var nameOK = regexp.MustCompile(`^[a-z0-9._-]+$`)

// Here makes (or reuses) the junction <root>\p\<name> to project and says
// where to build from: cd keeps cwd's place inside the project. A project of
// "" means cwd. Everything it creates is recorded in st.
func Here(project, cwd, root, name string, st *State) (link, cd string, err error) {
	if err := ValidateRoot(root); err != nil {
		return "", "", err
	}
	if project == "" {
		project = cwd
	}
	if isASCII(project) {
		return "", "", ErrAlreadyASCII
	}
	if name == "" {
		name = translit.Name(filepath.Base(project))
	} else if !nameOK.MatchString(name) {
		return "", "", fmt.Errorf("--name %q: use a-z, 0-9, dot, dash or underscore", name)
	}
	dir := filepath.Join(root, "p")
	if err := ensureDir(dir, st); err != nil {
		return "", "", err
	}
	for i := 1; ; i++ {
		candidate := name
		if i > 1 {
			candidate = fmt.Sprintf("%s-%d", name, i)
		}
		link = filepath.Join(dir, candidate)
		if target, err := win.JunctionTarget(link); err == nil && samePath(target, project) {
			break // ours already
		}
		if _, err := os.Lstat(link); os.IsNotExist(err) {
			if err := win.CreateJunction(link, project); err != nil {
				return "", "", err
			}
			st.Changes = append(st.Changes, Change{Kind: "junction", Path: link, Target: project, At: time.Now().UTC()})
			break
		}
		if i >= 99 {
			return "", "", errors.New("no free name under " + dir)
		}
	}
	cd = link
	if rel, err := filepath.Rel(project, cwd); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
		cd = filepath.Join(link, rel)
	}
	return link, cd, nil
}

// ValidateRoot accepts only a full, plain-ASCII path on a drive with no
// spaces — the folder exists to be safe for every tool.
func ValidateRoot(root string) error {
	if strings.HasPrefix(root, `\\`) || !filepath.IsAbs(root) || filepath.VolumeName(root) == "" {
		return fmt.Errorf("--root %q: use a full path on a drive, like C:\\dev", root)
	}
	if !isASCII(root) || strings.ContainsAny(root, " \t") {
		return fmt.Errorf("--root %q: the path must be plain ASCII with no spaces", root)
	}
	return nil
}

// ensureDir makes path and any missing parents, recording each one it made.
func ensureDir(path string, st *State) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if parent := filepath.Dir(path); parent != path {
		if err := ensureDir(parent, st); err != nil {
			return err
		}
	}
	if err := os.Mkdir(path, 0o755); err != nil {
		return err
	}
	st.Changes = append(st.Changes, Change{Kind: "dir", Path: path, At: time.Now().UTC()})
	return nil
}

func samePath(a, b string) bool { return strings.EqualFold(filepath.Clean(a), filepath.Clean(b)) }

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}
