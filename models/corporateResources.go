package models

type CorporateCatalog struct {
	Contexts []CorporateContext `yaml:"contexts" json:"contexts"`
}

type CorporateContext struct {
	ID          string              `yaml:"id" json:"id"`
	Name        string              `yaml:"name" json:"name"`
	Description string              `yaml:"description,omitempty" json:"description,omitempty"`
	Resources   []CorporateResource `yaml:"resources" json:"resources"`
}

type CorporateResource struct {
	ID             string       `yaml:"id" json:"id"`
	Name           string       `yaml:"name" json:"name"`
	Type           string       `yaml:"type" json:"type"`
	Description    string       `yaml:"description,omitempty" json:"description,omitempty"`
	Contexts       []string     `yaml:"contexts" json:"contexts"`
	AccessGroups   []string     `yaml:"accessGroups,omitempty" json:"accessGroups,omitempty"`
	InstructionURL string       `yaml:"instructionUrl,omitempty" json:"instructionUrl,omitempty"`
	Launch         LaunchConfig `yaml:"launch" json:"launch"`
}
