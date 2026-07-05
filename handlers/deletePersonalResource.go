package handlers

import (
	"catalog-agent/models"
	"encoding/json"
	"net/http"
	"strings"
)

func (h *CatalogHandler) DeletePersonalResource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.DeletePersonalResourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Invalid delete personal resource request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ResourceID == "" {
		http.Error(w, "resourceId is required", http.StatusBadRequest)
		return
	}

	// Get user from request context
	userUPN := r.Header.Get("X-User-UPN")
	if userUPN == "" {
		userUPN = "unknown@local"
	}

	h.logger.Info("Deleting personal resource: %s for user: %s", req.ResourceID, userUPN)

	// Load existing personal resources
	var catalog models.PersonalCatalog
	if err := h.loadPersonalResourcesFile(&catalog); err != nil {
		h.logger.Error("Failed to load personal resources: %v", err)
		http.Error(w, "Failed to delete resource", http.StatusInternalServerError)
		return
	}

	// Find and delete resource
	found := false
	newResources := []models.PersonalResource{}

	for _, res := range catalog.Resources {
		if res.ID == req.ResourceID && strings.EqualFold(res.Owner, userUPN) {
			found = true
			continue // Skip this resource (delete)
		}
		newResources = append(newResources, res)
	}

	if !found {
		http.Error(w, "Resource not found or not owned by user", http.StatusNotFound)
		return
	}

	catalog.Resources = newResources

	if err := h.savePersonalResourcesFile(catalog); err != nil {
		h.logger.Error("Failed to save personal resources after deletion: %v", err)
		http.Error(w, "Failed to delete resource", http.StatusInternalServerError)
		return
	}

	// Clear cache for user
	h.clearUserCache(userUPN)

	h.logger.Info("Personal resource deleted: %s for user: %s", req.ResourceID, userUPN)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "deleted",
		"id":     req.ResourceID,
	})
}
