package templates

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAutoDiscoverTechStack(t *testing.T) {
	// Test with Go project (current directory should have go.mod)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	// Go back to the project root
	projectRoot := filepath.Join(wd, "..", "..")

	stack, err := AutoDiscoverTechStack(projectRoot)
	if err != nil {
		t.Fatalf("AutoDiscoverTechStack failed: %v", err)
	}

	// Should detect Go project
	if stack.Language != "Go" {
		t.Errorf("Expected Language=Go, got %s", stack.Language)
	}

	if !stack.HasGoMod {
		t.Error("Expected HasGoMod=true")
	}

	if stack.BuildTool != "go build" {
		t.Errorf("Expected BuildTool=go build, got %s", stack.BuildTool)
	}
}

func TestAutoDiscoverArchitecture(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	projectRoot := filepath.Join(wd, "..", "..")

	arch, err := AutoDiscoverArchitecture(projectRoot)
	if err != nil {
		t.Fatalf("AutoDiscoverArchitecture failed: %v", err)
	}

	// Should detect CLI pattern (shiki-cli project)
	if !arch.HasCLI {
		t.Error("Expected HasCLI=true for CLI project")
	}

	// Should find main files
	if len(arch.MainFiles) == 0 {
		t.Error("Expected to find main files")
	}

	// Pattern should be detected
	if arch.Pattern == "" {
		t.Error("Expected architecture pattern to be detected")
	}
}

func TestAutoDiscoverTestingSetup(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	projectRoot := filepath.Join(wd, "..", "..")

	setup, err := AutoDiscoverTestingSetup(projectRoot)
	if err != nil {
		t.Fatalf("AutoDiscoverTestingSetup failed: %v", err)
	}

	// Should detect Go testing
	if setup.Framework != "Go testing" {
		t.Errorf("Expected Framework=Go testing, got %s", setup.Framework)
	}

	if !setup.HasUnitTests {
		t.Error("Expected HasUnitTests=true")
	}

	if setup.TestPattern != "*_test.go" {
		t.Errorf("Expected TestPattern=*_test.go, got %s", setup.TestPattern)
	}
}

func TestAutoDiscoverBuildSystem(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	projectRoot := filepath.Join(wd, "..", "..")

	build, err := AutoDiscoverBuildSystem(projectRoot)
	if err != nil {
		t.Fatalf("AutoDiscoverBuildSystem failed: %v", err)
	}

	// Should detect Go as build tool
	if build.BuildTool != "Go" {
		t.Errorf("Expected BuildTool=Go, got %s", build.BuildTool)
	}

	// Should have some config files
	if len(build.ConfigFiles) == 0 {
		t.Error("Expected to find config files")
	}
}

func TestHelperFunctions(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	projectRoot := filepath.Join(wd, "..", "..")

	// Test hasFile
	if !hasFile(projectRoot, "go.mod") {
		t.Error("Expected go.mod to exist")
	}

	// Test hasDir
	if !hasDir(projectRoot, "cmd") {
		t.Error("Expected cmd directory to exist")
	}

	// Test hasPattern (should find some Go patterns)
	if !hasPattern(projectRoot, "package ") {
		t.Error("Expected to find 'package ' pattern in Go files")
	}
}

func TestDiscoveryWithEmptyProject(t *testing.T) {
	// Create temporary empty directory
	tempDir, err := os.MkdirTemp("", "empty-project")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Test tech stack discovery with empty project
	stack, err := AutoDiscoverTechStack(tempDir)
	if err != nil {
		t.Fatalf("AutoDiscoverTechStack failed: %v", err)
	}

	// Should have empty/default values
	if stack.Language != "" {
		t.Errorf("Expected empty Language, got %s", stack.Language)
	}

	// Test architecture discovery with empty project
	arch, err := AutoDiscoverArchitecture(tempDir)
	if err != nil {
		t.Fatalf("AutoDiscoverArchitecture failed: %v", err)
	}

	if arch.Pattern != "Library/Package" {
		t.Errorf("Expected Pattern=Library/Package for empty project, got %s", arch.Pattern)
	}
}