package templates

import (
	"os"
	"path/filepath"
	"strings"
)

// TechStack represents the discovered technology stack
type TechStack struct {
	Language    string
	Framework   string
	Database    string
	BuildTool   string
	PackageMgr  string
	HasGoMod    bool
	HasPkgJSON  bool
	HasReqsTxt  bool
	HasGemfile  bool
	HasCargoTml bool
}

// Architecture represents discovered architectural patterns
type Architecture struct {
	Pattern     string // MVC, Clean, Layered, etc.
	MainFiles   []string
	EntryPoints []string
	HasAPI      bool
	HasWeb      bool
	HasCLI      bool
	HasDB       bool
}

// TestingSetup represents the testing framework and patterns
type TestingSetup struct {
	Framework      string
	TestDirs       []string
	TestPattern    string
	HasUnitTests   bool
	HasIntegTests  bool
	HasE2ETests    bool
	CoverageTools  []string
}

// BuildSystem represents build and deployment setup
type BuildSystem struct {
	BuildTool     string
	CIFiles       []string
	DockerFiles   []string
	ConfigFiles   []string
	DeployConfigs []string
}

// AutoDiscoverTechStack analyzes project for tech stack
func AutoDiscoverTechStack(projectRoot string) (TechStack, error) {
	stack := TechStack{}

	// Check for common dependency files
	files, err := os.ReadDir(projectRoot)
	if err != nil {
		return stack, err
	}

	for _, file := range files {
		name := file.Name()
		switch name {
		case "go.mod":
			stack.Language = "Go"
			stack.HasGoMod = true
			stack.BuildTool = "go build"
			stack.PackageMgr = "go mod"
		case "package.json":
			stack.Language = "JavaScript/TypeScript"
			stack.HasPkgJSON = true
			stack.BuildTool = "npm/yarn"
			stack.PackageMgr = "npm/yarn"
		case "requirements.txt":
			stack.Language = "Python"
			stack.HasReqsTxt = true
			stack.PackageMgr = "pip"
		case "Gemfile":
			stack.Language = "Ruby"
			stack.HasGemfile = true
			stack.PackageMgr = "bundler"
		case "Cargo.toml":
			stack.Language = "Rust"
			stack.HasCargoTml = true
			stack.BuildTool = "cargo"
			stack.PackageMgr = "cargo"
		}
	}

	// Try to detect framework if Go project
	if stack.Language == "Go" {
		if hasPattern(projectRoot, "gin.Engine") || hasPattern(projectRoot, "gin.Default") {
			stack.Framework = "Gin"
		} else if hasPattern(projectRoot, "fiber.New") {
			stack.Framework = "Fiber"
		} else if hasPattern(projectRoot, "echo.New") {
			stack.Framework = "Echo"
		} else if hasPattern(projectRoot, "mux.NewRouter") {
			stack.Framework = "Gorilla Mux"
		}
	}

	// Detect database usage
	if hasPattern(projectRoot, "database/sql") || hasPattern(projectRoot, "gorm") {
		stack.Database = "SQL"
	} else if hasPattern(projectRoot, "mongo") {
		stack.Database = "MongoDB"
	} else if hasPattern(projectRoot, "redis") {
		stack.Database = "Redis"
	}

	return stack, nil
}

// AutoDiscoverArchitecture identifies architectural patterns
func AutoDiscoverArchitecture(projectRoot string) (Architecture, error) {
	arch := Architecture{
		MainFiles:   make([]string, 0),
		EntryPoints: make([]string, 0),
	}

	// Find main entry points
	err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue on errors
		}

		if info.IsDir() {
			return nil
		}

		name := info.Name()

		// Check for main files
		if name == "main.go" || strings.HasPrefix(name, "main.") {
			arch.MainFiles = append(arch.MainFiles, path)
			arch.EntryPoints = append(arch.EntryPoints, path)
		}

		// Check for CLI patterns
		if strings.Contains(path, "cmd/") && strings.HasSuffix(name, ".go") {
			arch.HasCLI = true
		}

		// Check for API patterns
		if strings.Contains(path, "api/") || strings.Contains(path, "handler") ||
		   strings.Contains(path, "controller") || strings.Contains(path, "router") {
			arch.HasAPI = true
		}

		// Check for web patterns
		if strings.Contains(path, "web/") || strings.Contains(path, "static/") ||
		   strings.Contains(path, "templates/") || strings.Contains(path, "views/") {
			arch.HasWeb = true
		}

		return nil
	})

	if err != nil {
		return arch, err
	}

	// Determine architecture pattern
	if arch.HasAPI && arch.HasWeb {
		arch.Pattern = "Full-Stack Web"
	} else if arch.HasAPI {
		arch.Pattern = "API/Microservice"
	} else if arch.HasCLI {
		arch.Pattern = "CLI Application"
	} else if len(arch.MainFiles) > 0 {
		arch.Pattern = "Simple Application"
	} else {
		arch.Pattern = "Library/Package"
	}

	return arch, nil
}

// AutoDiscoverTestingSetup finds testing framework and patterns
func AutoDiscoverTestingSetup(projectRoot string) (TestingSetup, error) {
	setup := TestingSetup{
		TestDirs:      make([]string, 0),
		CoverageTools: make([]string, 0),
	}

	// Walk project to find test files
	err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			name := info.Name()
			if name == "test" || name == "tests" || name == "__tests__" {
				setup.TestDirs = append(setup.TestDirs, path)
			}
			return nil
		}

		name := info.Name()

		// Go test patterns
		if strings.HasSuffix(name, "_test.go") {
			setup.Framework = "Go testing"
			setup.TestPattern = "*_test.go"
			setup.HasUnitTests = true

			// Check for integration tests
			if strings.Contains(name, "integration") || strings.Contains(path, "integration") {
				setup.HasIntegTests = true
			}

			// Check for e2e tests
			if strings.Contains(name, "e2e") || strings.Contains(path, "e2e") {
				setup.HasE2ETests = true
			}
		}

		// JavaScript test patterns
		if strings.Contains(name, ".test.") || strings.Contains(name, ".spec.") {
			if strings.Contains(name, ".js") || strings.Contains(name, ".ts") {
				setup.Framework = "Jest/Mocha"
				setup.TestPattern = "*.test.* / *.spec.*"
				setup.HasUnitTests = true
			}
		}

		return nil
	})

	// Check for coverage tools
	if hasFile(projectRoot, "coverage") {
		setup.CoverageTools = append(setup.CoverageTools, "coverage")
	}

	return setup, err
}

// AutoDiscoverBuildSystem identifies build/deploy setup
func AutoDiscoverBuildSystem(projectRoot string) (BuildSystem, error) {
	build := BuildSystem{
		CIFiles:       make([]string, 0),
		DockerFiles:   make([]string, 0),
		ConfigFiles:   make([]string, 0),
		DeployConfigs: make([]string, 0),
	}

	files, err := os.ReadDir(projectRoot)
	if err != nil {
		return build, err
	}

	for _, file := range files {
		name := file.Name()

		// Build tools
		switch name {
		case "Makefile":
			build.BuildTool = "Make"
		case "go.mod":
			build.BuildTool = "Go"
		case "package.json":
			build.BuildTool = "npm/yarn"
		case "Cargo.toml":
			build.BuildTool = "Cargo"
		}

		// Docker files
		if strings.HasPrefix(name, "Dockerfile") || name == "docker-compose.yml" {
			build.DockerFiles = append(build.DockerFiles, name)
		}

		// CI files
		if strings.Contains(name, "github") || name == ".gitlab-ci.yml" ||
		   name == "Jenkinsfile" || strings.Contains(name, "action") {
			build.CIFiles = append(build.CIFiles, name)
		}

		// Config files
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") ||
		   strings.HasSuffix(name, ".toml") || strings.HasSuffix(name, ".json") {
			build.ConfigFiles = append(build.ConfigFiles, name)
		}
	}

	// Check for CI directories
	if hasDir(projectRoot, ".github") {
		build.CIFiles = append(build.CIFiles, ".github/")
	}

	// Check for deployment configs
	if hasDir(projectRoot, "deploy") || hasDir(projectRoot, "k8s") || hasDir(projectRoot, "helm") {
		build.DeployConfigs = append(build.DeployConfigs, "deploy/", "k8s/", "helm/")
	}

	return build, nil
}

// Helper functions

func hasPattern(projectRoot, pattern string) bool {
	found := false
	filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			content, err := os.ReadFile(path)
			if err == nil && strings.Contains(string(content), pattern) {
				found = true
				return filepath.SkipDir // Stop walking
			}
		}
		return nil
	})
	return found
}

func hasFile(projectRoot, filename string) bool {
	_, err := os.Stat(filepath.Join(projectRoot, filename))
	return err == nil
}

func hasDir(projectRoot, dirname string) bool {
	info, err := os.Stat(filepath.Join(projectRoot, dirname))
	return err == nil && info.IsDir()
}