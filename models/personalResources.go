package models

type PersonalCatalog struct {
	Resources []PersonalResource `yaml:"resources" json:"resources"`
}

type PersonalResource struct {
	ID          string       `yaml:"id" json:"id"`
	Name        string       `yaml:"name" json:"name"`
	Type        string       `yaml:"type" json:"type"`
	Description string       `yaml:"description,omitempty" json:"description,omitempty"`
	Contexts    []string     `yaml:"contexts" json:"contexts"`
	Owner       string       `yaml:"owner" json:"owner"`
	Launch      LaunchConfig `yaml:"launch" json:"launch"`
}

// PersonalResourceCreate is used for API input
type PersonalResourceCreate struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Type        string       `json:"type"`
	Description string       `json:"description,omitempty"`
	Contexts    []string     `json:"contexts"`
	Launch      LaunchConfig `json:"launch"`
}
