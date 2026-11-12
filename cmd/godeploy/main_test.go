package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIVersion(t *testing.T) {
	if appName != "GoDeploy" {
		t.Errorf("Expected GoDeploy, got %s", appName)
	}

	if appVersion != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", appVersion)
	}
}

func TestCLIConfiguration(t *testing.T) {
	if imageName != "godeploy-app:v1" {
		t.Errorf("Unexpected image name: %s", imageName)
	}

	if containerName != "godeploy-app" {
		t.Errorf("Unexpected container name: %s", containerName)
	}

	if appPort != "8080" {
		t.Errorf("Unexpected application port: %s", appPort)
	}
}

func TestPrintUsage(t *testing.T) {
	output := captureOutput(t, printUsage)

	expectedCommands := []string{
		"version",
		"doctor",
		"build",
		"deploy",
		"status",
		"history",
		"down",
	}

	for _, command := range expectedCommands {
		if !strings.Contains(output, command) {
			t.Errorf("Usage output does not contain command: %s", command)
		}
	}
}

func TestRunPodmanSuccess(t *testing.T) {
	setupMockPodman(t, "Write-Output 'mock podman success'")

	err := runPodman(context.Background(), "version")

	if err != nil {
		t.Fatalf("Expected successful command, got error: %v", err)
	}
}

func TestRunPodmanFailure(t *testing.T) {
	setupMockPodman(t, "Write-Error 'mock podman failure'; exit 1")

	err := runPodman(context.Background(), "invalid-command")

	if err == nil {
		t.Fatal("Expected an error, got nil")
	}

	if !strings.Contains(err.Error(), "podman invalid-command") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestPodmanOutputSuccess(t *testing.T) {
	setupMockPodman(t, "Write-Output 'mock container output'")

	output, err := podmanOutput(context.Background(), "ps")

	if err != nil {
		t.Fatalf("Expected successful command, got error: %v", err)
	}

	if !strings.Contains(output, "mock container output") {
		t.Errorf("Unexpected command output: %s", output)
	}
}

func TestPodmanOutputFailure(t *testing.T) {
	setupMockPodman(t, "Write-Error 'mock failure'; exit 1")

	output, err := podmanOutput(context.Background(), "invalid-command")

	if err == nil {
		t.Fatal("Expected an error, got nil")
	}

	if output != "" {
		t.Errorf("Expected empty output on failure, got: %s", output)
	}
}

func captureOutput(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout

	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("Could not create output pipe: %v", err)
	}

	os.Stdout = writePipe

	fn()

	writePipe.Close()
	os.Stdout = oldStdout

	output, err := io.ReadAll(readPipe)
	if err != nil {
		t.Fatalf("Could not read captured output: %v", err)
	}

	readPipe.Close()

	return string(output)
}

func setupMockPodman(t *testing.T, script string) {
	t.Helper()

	tempDir := t.TempDir()

	mockPath := filepath.Join(tempDir, "podman.cmd")

	mockScript := "@echo off\r\npowershell.exe -NoProfile -Command \"" +
		script + "\"\r\n"

	if err := os.WriteFile(mockPath, []byte(mockScript), 0600); err != nil {
		t.Fatalf("Could not create mock Podman: %v", err)
	}

	oldPath := os.Getenv("PATH")

	t.Setenv("PATH", tempDir+string(os.PathListSeparator)+oldPath)
	t.Setenv("PATHEXT", ".COM;.EXE;.BAT;.CMD")
}
