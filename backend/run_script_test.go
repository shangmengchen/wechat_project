package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunShHelpUnderDash(t *testing.T) {
	dash, ok := findDash()
	if !ok {
		t.Skip("dash not available")
	}

	cmd := exec.Command(dash, "./run.sh", "--help")
	cmd.Dir = "."
	cmd.Env = withShellPath(dash)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected help command to succeed, got error: %v\noutput:\n%s", err, output)
	}
	if !strings.Contains(string(output), "Usage: ./run.sh [options]") {
		t.Fatalf("expected usage output, got:\n%s", output)
	}
}

func TestRunShBuildAliasUnderDash(t *testing.T) {
	dash, ok := findDash()
	if !ok {
		t.Skip("dash not available")
	}

	cmd := exec.Command(dash, "./run.sh", "build", "--help")
	cmd.Dir = "."
	cmd.Env = withShellPath(dash)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected build alias help command to succeed, got error: %v\noutput:\n%s", err, output)
	}
	if !strings.Contains(string(output), "Usage: ./run.sh [options]") {
		t.Fatalf("expected usage output for build alias, got:\n%s", output)
	}
}

func findDash() (string, bool) {
	candidates := []string{"dash"}
	if runtime.GOOS == "windows" {
		candidates = append(candidates,
			`D:\Git\Git\usr\bin\dash.exe`,
			`C:\Program Files\Git\usr\bin\dash.exe`,
		)
	}

	for _, candidate := range candidates {
		if filepath.IsAbs(candidate) {
			if _, err := os.Stat(candidate); err == nil {
				return candidate, true
			}
			continue
		}
		if resolved, err := exec.LookPath(candidate); err == nil {
			return resolved, true
		}
	}
	return "", false
}

func withShellPath(shellPath string) []string {
	env := os.Environ()
	shellDir := filepath.Dir(shellPath)
	currentPath := os.Getenv("PATH")
	prefixedPath := shellDir + string(os.PathListSeparator) + currentPath

	found := false
	for idx, entry := range env {
		if strings.HasPrefix(entry, "PATH=") {
			env[idx] = "PATH=" + prefixedPath
			found = true
			break
		}
	}
	if !found {
		env = append(env, "PATH="+prefixedPath)
	}
	return env
}
