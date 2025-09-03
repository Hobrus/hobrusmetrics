package buildinfo

import "fmt"

// Build information is injected via -ldflags at build time.
var Version string
var Date string
var Commit string

func normalize(s string) string {
	if s == "" {
		return "N/A"
	}
	return s
}

// Print выводит информацию о сборке в stdout.
func Print(version, date, commit string) {
	fmt.Printf("Build version: %s\n", normalize(version))
	fmt.Printf("Build date: %s\n", normalize(date))
	fmt.Printf("Build commit: %s\n", normalize(commit))
}

// PrintSelf выводит информацию из пакетных переменных Version/Date/Commit.
func PrintSelf() {
	Print(Version, Date, Commit)
}
