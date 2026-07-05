package handlers

import (
	"catalog-agent/models"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func (h *CatalogHandler) AddPersonalResource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.AddPersonalResourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Invalid add personal resource request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.ID == "" || req.Name == "" || req.Type == "" {
		http.Error(w, "id, name, and type are required", http.StatusBadRequest)
		return
	}

	// Get user from request context (set by auth middleware)
	userUPN := r.Header.Get("X-User-UPN")
	if userUPN == "" {
		userUPN = "unknown@local" // Fallback
	}

	h.logger.Info("Adding personal resource: %s for user: %s", req.ID, userUPN)

	// Create personal resource
	personalRes := models.PersonalResource{
		ID:          req.ID,
		Name:        req.Name,
		Type:        req.Type,
		Description: req.Description,
		Contexts:    req.Contexts,
		Owner:       userUPN,
		Launch:      req.Launch,
	}

	// Load existing personal resources
	var catalog models.PersonalCatalog
	if err := h.loadPersonalResourcesFile(&catalog); err != nil && !os.IsNotExist(err) {
		h.logger.Error("Failed to load personal resources: %v", err)
		http.Error(w, "Failed to save resource", http.StatusInternalServerError)
		return
	}

	// Check if resource already exists
	for i, res := range catalog.Resources {
		if res.ID == req.ID && strings.EqualFold(res.Owner, userUPN) {
			// Update existing
			catalog.Resources[i] = personalRes
			if err := h.savePersonalResourcesFile(catalog); err != nil {
				h.logger.Error("Failed to update personal resource: %v", err)
				http.Error(w, "Failed to update resource", http.StatusInternalServerError)
				return
			}

			// Clear cache for user
			h.clearUserCache(userUPN)

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "updated",
				"id":     req.ID,
			})
			return
		}
	}

	// Add new resource
	catalog.Resources = append(catalog.Resources, personalRes)

	if err := h.savePersonalResourcesFile(catalog); err != nil {
		h.logger.Error("Failed to save personal resource: %v", err)
		http.Error(w, "Failed to save resource", http.StatusInternalServerError)
		return
	}

	// Clear cache for user
	h.clearUserCache(userUPN)

	h.logger.Info("Personal resource added: %s for user: %s", req.ID, userUPN)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "created",
		"id":     req.ID,
	})
}

func (h *CatalogHandler) loadPersonalResourcesFile(catalog *models.PersonalCatalog) error {
	data, err := os.ReadFile(h.resourceLoader.personalPath)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, catalog)
}

func (h *CatalogHandler) savePersonalResourcesFile(catalog models.PersonalCatalog) error {
	data, err := yaml.Marshal(catalog)
	if err != nil {
		return err
	}
	return os.WriteFile(h.resourceLoader.personalPath, data, 0644)
}

func (h *CatalogHandler) clearUserCache(userUPN string) {
	h.cacheMutex.Lock()
	defer h.cacheMutex.Unlock()

	userUPN = strings.ToLower(strings.TrimSpace(userUPN))
	delete(h.cache, userUPN)
}
