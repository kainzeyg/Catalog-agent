package entities

import "time"

type Resource struct {
	ID             string
	Name           string
	Type           string
	Description    string
	Contexts       []string
	InstructionURL string
	AccessGroups   []string
	Launch         LaunchConfig
	Available      bool
	Source         string // "corporate" or "personal"
	Owner          string
	LastChecked    time.Time
}

type LaunchConfig struct {
	Executable string
	Args       []string
	URL        string
	Path       string
}

type Context struct {
	ID          string
	Name        string
	Description string
	Resources   []Resource
}

type FilteredCatalog struct {
	Contexts []Context
	UserUPN  string
	Groups   []string
}
