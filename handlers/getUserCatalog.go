package handlers

import (
	"catalog-agent/audit"
	"catalog-agent/entities"
	"catalog-agent/models"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type CatalogHandler struct {
	logger         *audit.Logger
	adClient       *ADClient
	resourceLoader *ResourceLoader
	cache          map[string]*entities.FilteredCatalogData
	cacheMutex     sync.RWMutex
}

func NewCatalogHandler(logger *audit.Logger, adClient *ADClient, resourceLoader *ResourceLoader) *CatalogHandler {
	return &CatalogHandler{
		logger:         logger,
		adClient:       adClient,
		resourceLoader: resourceLoader,
		cache:          make(map[string]*entities.FilteredCatalogData),
	}
}

func (h *CatalogHandler) GetUserCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.CatalogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Invalid catalog request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserUPN == "" {
		http.Error(w, "user field is required", http.StatusBadRequest)
		return
	}

	h.logger.Info("Catalog request for user: %s", req.UserUPN)

	// Check cache
	userUPN := strings.ToLower(strings.TrimSpace(req.UserUPN))
	h.cacheMutex.RLock()
	if cached, exists := h.cache[userUPN]; exists {
		// Cache valid for 5 minutes
		if time.Since(cached.GeneratedAt) < 5*time.Minute {
			h.cacheMutex.RUnlock()
			h.logger.Debug("Returning cached catalog for user: %s", userUPN)
			h.sendCatalogResponse(w, cached)
			return
		}
	}
	h.cacheMutex.RUnlock()

	// Get user groups from AD
	userGroups, err := h.adClient.GetUserGroups(req.UserUPN)
	if err != nil {
		h.logger.Warn("AD unavailable, returning all resources for user: %s, error: %v", req.UserUPN, err)
		userGroups = []string{} // Empty groups = all resources accessible
	}

	// Load corporate resources
	corporateCatalog, err := h.resourceLoader.LoadCorporateResources()
	if err != nil {
		h.logger.Error("Failed to load corporate resources: %v", err)
		http.Error(w, "Failed to load resources", http.StatusInternalServerError)
		return
	}

	// Load personal resources
	personalResources, err := h.resourceLoader.LoadPersonalResources(req.UserUPN)
	if err != nil {
		h.logger.Error("Failed to load personal resources: %v", err)
		personalResources = []models.PersonalResource{} // Continue with empty personal
	}

	// Build filtered catalog
	filteredCatalog := h.buildFilteredCatalog(corporateCatalog, personalResources, userGroups, req.UserUPN)

	// Update cache
	h.cacheMutex.Lock()
	h.cache[userUPN] = filteredCatalog
	h.cacheMutex.Unlock()

	h.logger.Info("Catalog built for user: %s, contexts: %d", req.UserUPN, len(filteredCatalog.Contexts))
	h.sendCatalogResponse(w, filteredCatalog)
}

func (h *CatalogHandler) buildFilteredCatalog(
	corporate *models.CorporateCatalog,
	personal []models.PersonalResource,
	userGroups []string,
	userUPN string,
) *entities.FilteredCatalogData {

	filteredData := &entities.FilteredCatalogData{
		UserUPN:     userUPN,
		UserGroups:  userGroups,
		Contexts:    []entities.FilteredContext{},
		GeneratedAt: time.Now(),
	}

	// Build context map
	contextMap := make(map[string]*entities.FilteredContext)

	// Add corporate contexts
	for _, corpCtx := range corporate.Contexts {
		contextMap[corpCtx.ID] = &entities.FilteredContext{
			ID:          corpCtx.ID,
			Name:        corpCtx.Name,
			Description: corpCtx.Description,
			Resources:   []entities.FilteredResource{},
		}
	}

	// Process corporate resources
	for _, corpCtx := range corporate.Contexts {
		for _, corpRes := range corpCtx.Resources {
			// Check if user has access
			if h.hasAccess(corpRes.AccessGroups, userGroups) {
				filteredRes := entities.FilteredResource{
					ID:             corpRes.ID,
					Name:           corpRes.Name,
					Type:           corpRes.Type,
					InstructionURL: corpRes.InstructionURL,
					Source:         "corporate",
				}

				// Check availability
				filteredRes.Available = h.checkResourceAvailability(corpRes.Type, corpRes.Launch)

				// Add to appropriate contexts
				for _, ctxID := range corpRes.Contexts {
					if ctx, exists := contextMap[ctxID]; exists {
						ctx.Resources = append(ctx.Resources, filteredRes)
					}
				}
			}
		}
	}

	// Process personal resources
	for _, persRes := range personal {
		// Only show personal resources to their owner
		if strings.EqualFold(persRes.Owner, userUPN) {
			filteredRes := entities.FilteredResource{
				ID:     persRes.ID,
				Name:   persRes.Name,
				Type:   persRes.Type,
				Source: "personal",
				Owner:  persRes.Owner,
			}

			// Check availability
			filteredRes.Available = h.checkResourceAvailability(persRes.Type, persRes.Launch)

			// Add to contexts (create if doesn't exist)
			for _, ctxID := range persRes.Contexts {
				if ctx, exists := contextMap[ctxID]; exists {
					ctx.Resources = append(ctx.Resources, filteredRes)
				} else {
					// Create new context for personal resources
					contextMap[ctxID] = &entities.FilteredContext{
						ID:          ctxID,
						Name:        ctxID,
						Description: "Personal resources",
						Resources:   []entities.FilteredResource{filteredRes},
					}
				}
			}
		}
	}

	// Convert map to slice
	for _, ctx := range contextMap {
		if len(ctx.Resources) > 0 {
			filteredData.Contexts = append(filteredData.Contexts, *ctx)
		}
	}

	return filteredData
}

func (h *CatalogHandler) hasAccess(resourceGroups []string, userGroups []string) bool {
	// If no groups specified, accessible to all
	if len(resourceGroups) == 0 {
		return true
	}

	// If AD is not available, return true for all
	if len(userGroups) == 0 {
		return true
	}

	// Check if user has any of the required groups
	groupMap := make(map[string]bool)
	for _, g := range userGroups {
		groupMap[strings.ToLower(strings.TrimSpace(g))] = true
	}

	for _, requiredGroup := range resourceGroups {
		if groupMap[strings.ToLower(strings.TrimSpace(requiredGroup))] {
			return true
		}
	}

	return false
}

func (h *CatalogHandler) checkResourceAvailability(resType string, launch models.LaunchConfig) bool {
	switch strings.ToLower(resType) {
	case "application":
		if launch.Executable == "" {
			return false
		}
		_, err := os.Stat(launch.Executable)
		return err == nil
	case "web":
		return true // Always available (DNS/HTTP check could be added)
	case "folder":
		if launch.Path == "" {
			return false
		}
		info, err := os.Stat(launch.Path)
		return err == nil && info.IsDir()
	case "file":
		if launch.Path == "" {
			return false
		}
		info, err := os.Stat(launch.Path)
		return err == nil && !info.IsDir()
	default:
		return false
	}
}

func (h *CatalogHandler) sendCatalogResponse(w http.ResponseWriter, filtered *entities.FilteredCatalogData) {
	response := models.CatalogResponse{
		Contexts: []models.ContextResponse{},
	}

	for _, ctx := range filtered.Contexts {
		ctxResp := models.ContextResponse{
			ID:        ctx.ID,
			Name:      ctx.Name,
			Resources: []models.ResourceResponse{},
		}

		for _, res := range ctx.Resources {
			resp := models.ResourceResponse{
				ID:             res.ID,
				Name:           res.Name,
				Type:           res.Type,
				Available:      res.Available,
				InstructionURL: res.InstructionURL,
				Icon:           res.Icon,
			}
			ctxResp.Resources = append(ctxResp.Resources, resp)
		}

		response.Contexts = append(response.Contexts, ctxResp)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
