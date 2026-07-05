package models

// API Request/Response models

type CatalogRequest struct {
	UserUPN string `json:"user"` // User Principal Name
}

type CatalogResponse struct {
	Contexts []ContextResponse `json:"contexts"`
}

type ContextResponse struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	Resources []ResourceResponse `json:"resources"`
}

type ResourceResponse struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	Available      bool   `json:"available"`
	InstructionURL string `json:"instructionUrl,omitempty"`
	Icon           string `json:"icon,omitempty"`
}

type ExecuteRequest struct {
	ResourceID string `json:"resourceId"`
}

type ExecuteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

type AddPersonalResourceRequest struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Type        string       `json:"type"` // application, web, folder, file
	Description string       `json:"description,omitempty"`
	Contexts    []string     `json:"contexts"`
	Launch      LaunchConfig `json:"launch"`
}

type LaunchConfig struct {
	Executable string   `json:"executable,omitempty"`
	Args       []string `json:"args,omitempty"`
	URL        string   `json:"url,omitempty"`
	Path       string   `json:"path,omitempty"`
}

type DeletePersonalResourceRequest struct {
	ResourceID string `json:"resourceId"`
}
