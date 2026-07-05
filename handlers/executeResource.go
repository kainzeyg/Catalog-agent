package handlers

import (
	"catalog-agent/entities"
	"catalog-agent/executor"
	"catalog-agent/models"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func (h *CatalogHandler) ExecuteResource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Invalid execute request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ResourceID == "" {
		http.Error(w, "resourceId is required", http.StatusBadRequest)
		return
	}

	userUPN := r.Header.Get("X-User-UPN")
	if userUPN == "" {
		userUPN = "unknown@local"
	}

	h.logger.Info("Execute request for resource: %s by user: %s", req.ResourceID, userUPN)

	// Find resource
	resource, err := h.findResource(req.ResourceID, userUPN)
	if err != nil {
		h.logger.Error("Resource not found: %s, error: %v", req.ResourceID, err)
		http.Error(w, "Resource not found", http.StatusNotFound)
		return
	}

	// Execute resource
	exec := executor.NewExecutor()
	if err := exec.Execute(*resource); err != nil {
		h.logger.Error("Failed to execute resource %s: %v", req.ResourceID, err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.ExecuteResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	h.logger.Info("Resource executed successfully: %s by user: %s", req.ResourceID, userUPN)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(models.ExecuteResponse{
		Success: true,
	})
}

func (h *CatalogHandler) findResource(resourceID string, userUPN string) (*entities.Resource, error) {
	// Get user groups for access check
	userGroups, err := h.adClient.GetUserGroups(userUPN)
	if err != nil {
		userGroups = []string{} // AD offline, allow all
	}

	// Load corporate resources
	corporateCatalog, err := h.resourceLoader.LoadCorporateResources()
	if err != nil {
		return nil, err
	}

	// Search in corporate resources
	for _, ctx := range corporateCatalog.Contexts {
		for _, res := range ctx.Resources {
			if res.ID == resourceID {
				// Check access
				if h.hasAccess(res.AccessGroups, userGroups) {
					return &entities.Resource{
						ID:          res.ID,
						Name:        res.Name,
						Type:        res.Type,
						Description: res.Description,
						Launch: entities.LaunchConfig{
							Executable: res.Launch.Executable,
							Args:       res.Launch.Args,
							URL:        res.Launch.URL,
							Path:       res.Launch.Path,
						},
						Available: h.checkResourceAvailability(res.Type, res.Launch),
						Source:    "corporate",
					}, nil
				}
				return nil, fmt.Errorf("access denied to resource")
			}
		}
	}

	// Search in personal resources
	var personalCatalog models.PersonalCatalog
	if err := h.loadPersonalResourcesFile(&personalCatalog); err == nil {
		for _, res := range personalCatalog.Resources {
			if res.ID == resourceID && strings.EqualFold(res.Owner, userUPN) {
				return &entities.Resource{
					ID:          res.ID,
					Name:        res.Name,
					Type:        res.Type,
					Description: res.Description,
					Launch: entities.LaunchConfig{
						Executable: res.Launch.Executable,
						Args:       res.Launch.Args,
						URL:        res.Launch.URL,
						Path:       res.Launch.Path,
					},
					Available: h.checkResourceAvailability(res.Type, res.Launch),
					Source:    "personal",
					Owner:     res.Owner,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("resource not found: %s", resourceID)
}
