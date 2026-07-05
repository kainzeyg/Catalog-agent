package entities

import "time"

type FilteredCatalogData struct {
	UserUPN     string
	UserGroups  []string
	Contexts    []FilteredContext
	GeneratedAt time.Time
}

type FilteredContext struct {
	ID          string
	Name        string
	Description string
	Resources   []FilteredResource
}

type FilteredResource struct {
	ID             string
	Name           string
	Type           string
	Available      bool
	InstructionURL string
	Icon           string
	Source         string // "corporate" or "personal"
	Owner          string
}
