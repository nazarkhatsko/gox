package console

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ShortPath replaces the home directory prefix of path with ~.
func ShortPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}

	if path == home {
		return "~"
	}

	if strings.HasPrefix(path, home+string(filepath.Separator)) {
		return "~" + path[len(home):]
	}

	return path
}

// Duration formats d compactly: 8ms, 1.4s, 3m 12s, 2h 5m, 3d 4h.
func Duration(d time.Duration) string {
	switch {
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	case d < 10*time.Second && d%time.Second != 0:
		return fmt.Sprintf("%.1fs", d.Seconds())
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	default:
		return fmt.Sprintf("%dd %dh", int(d.Hours())/24, int(d.Hours())%24)
	}
}

// Relative formats t relative to now: "in 3m 12s", "2m 5s ago". Times more
// than a day away are shown as a local date instead.
func Relative(t, now time.Time) string {
	d := t.Sub(now)
	if d > 24*time.Hour || d < -24*time.Hour {
		return t.Local().Format("2006-01-02 15:04")
	}

	if d >= 0 {
		if d < time.Second {
			return "now"
		}

		return "in " + Duration(d.Truncate(time.Second))
	}

	d = -d
	if d < time.Second {
		return "just now"
	}

	return Duration(d.Truncate(time.Second)) + " ago"
}
