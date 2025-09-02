package buildinfo

import "fmt"

// These variables are intended to be set via -ldflags -X at build time.
var (
	Version string
	Date    string
	Commit  string
)

func normalize(s string) string {
	if s == "" {
		return "N/A"
	}
	return s
}

// Print writes build information to stdout.
func Print() {
	fmt.Printf("Build version: %s\n", normalize(Version))
	fmt.Printf("Build date: %s\n", normalize(Date))
	fmt.Printf("Build commit: %s\n", normalize(Commit))
}
