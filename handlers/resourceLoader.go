package handlers

import (
	"catalog-agent/models"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type ResourceLoader struct {
	corporatePath string
	personalPath  string
}

func NewResourceLoader(corporatePath, personalPath string) *ResourceLoader {
	return &ResourceLoader{
		corporatePath: corporatePath,
		personalPath:  personalPath,
	}
}

func (r *ResourceLoader) LoadCorporateResources() (*models.CorporateCatalog, error) {
	data, err := os.ReadFile(r.corporatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read corporate resources: %w", err)
	}

	var catalog models.CorporateCatalog
	if err := yaml.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("failed to parse corporate resources: %w", err)
	}

	return &catalog, nil
}

func (r *ResourceLoader) LoadPersonalResources(userUPN string) ([]models.PersonalResource, error) {
	data, err := os.ReadFile(r.personalPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []models.PersonalResource{}, nil
		}
		return nil, fmt.Errorf("failed to read personal resources: %w", err)
	}

	var catalog models.PersonalCatalog
	if err := yaml.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("failed to parse personal resources: %w", err)
	}

	// Filter resources owned by user
	userResources := []models.PersonalResource{}
	for _, res := range catalog.Resources {
		if strings.EqualFold(res.Owner, userUPN) {
			userResources = append(userResources, res)
		}
	}

	return userResources, nil
}
