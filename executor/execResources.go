package executor

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"catalog-agent/entities"

	"github.com/skratchdot/open-golang/open"
)

type Executor struct{}

func NewExecutor() *Executor {
	return &Executor{}
}

func (e *Executor) Execute(resource entities.Resource) error {
	// Check if resource is available
	if !resource.Available {
		return fmt.Errorf("resource '%s' is not available", resource.Name)
	}

	switch strings.ToLower(resource.Type) {
	case "web":
		return e.executeWeb(resource.Launch.URL)
	case "application":
		return e.executeApplication(resource.Launch.Executable, resource.Launch.Args)
	case "folder":
		return e.executeFolder(resource.Launch.Path)
	case "file":
		return e.executeFile(resource.Launch.Path)
	default:
		return fmt.Errorf("unsupported resource type: %s", resource.Type)
	}
}

func (e *Executor) executeWeb(url string) error {
	if url == "" {
		return fmt.Errorf("URL is empty")
	}
	return open.Run(url)
}

func (e *Executor) executeApplication(exe string, args []string) error {
	if exe == "" {
		return fmt.Errorf("executable path is empty")
	}

	// Check if file exists
	if _, err := os.Stat(exe); err != nil {
		return fmt.Errorf("executable not found: %s", exe)
	}

	if runtime.GOOS == "windows" {
		// Build command with proper quoting for Windows
		cmdArgs := []string{"/C", "start", "\"\"", exe}
		if len(args) > 0 {
			cmdArgs = append(cmdArgs, args...)
		}
		cmd := exec.Command("cmd", cmdArgs...)
		return cmd.Start()
	}

	// Linux/Mac
	cmd := exec.Command(exe, args...)
	return cmd.Start()
}

func (e *Executor) executeFolder(path string) error {
	if path == "" {
		return fmt.Errorf("folder path is empty")
	}

	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("folder not found: %s", path)
	}

	if runtime.GOOS == "windows" {
		cmd := exec.Command("explorer", path)
		return cmd.Start()
	}

	return open.Run(path)
}

func (e *Executor) executeFile(path string) error {
	if path == "" {
		return fmt.Errorf("file path is empty")
	}

	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("file not found: %s", path)
	}

	return open.Run(path)
}
