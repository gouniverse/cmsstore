package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gouniverse/cmsstore"
)

// handleTranslationsEndpoint handles HTTP requests for the /api/translations endpoint
func (api *RestAPI) handleTranslationsEndpoint(w http.ResponseWriter, r *http.Request, pathParts []string) {
	switch r.Method {
	case http.MethodPost:
		// Create a new translation
		api.handleTranslationCreate(w, r)
	case http.MethodGet:
		// Get translation(s)
		if len(pathParts) > 0 && pathParts[0] != "" {
			// Get a specific translation by ID
			api.handleTranslationGet(w, r, pathParts[0])
		} else {
			// List all translations
			api.handleTranslationList(w, r)
		}
	case http.MethodPut:
		// Update a translation
		if len(pathParts) > 0 && pathParts[0] != "" {
			api.handleTranslationUpdate(w, r, pathParts[0])
		} else {
			http.Error(w, `{"success":false,"error":"Translation ID required for update"}`, http.StatusBadRequest)
		}
	case http.MethodDelete:
		// Delete a translation
		if len(pathParts) > 0 && pathParts[0] != "" {
			api.handleTranslationDelete(w, r, pathParts[0])
		} else {
			http.Error(w, `{"success":false,"error":"Translation ID required for deletion"}`, http.StatusBadRequest)
		}
	default:
		http.Error(w, `{"success":false,"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

// handleTranslationCreate handles HTTP requests to create a translation
func (api *RestAPI) handleTranslationCreate(w http.ResponseWriter, r *http.Request) {
	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"success":false,"error":"Failed to read request body: %v"}`, err), http.StatusBadRequest)
		return
	}

	// Parse the request body
	var translationData map[string]interface{}
	if err := json.Unmarshal(body, &translationData); err != nil {
		http.Error(w, fmt.Sprintf(`{"success":false,"error":"Failed to parse request body: %v"}`, err), http.StatusBadRequest)
		return
	}

	// Validate required fields
	key, ok := translationData["key"].(string)
	if !ok || key == "" {
		http.Error(w, `{"success":false,"error":"Key is required"}`, http.StatusBadRequest)
		return
	}

	locale, ok := translationData["locale"].(string)
	if !ok || locale == "" {
		http.Error(w, `{"success":false,"error":"Locale is required"}`, http.StatusBadRequest)
		return
	}

	text, ok := translationData["text"].(string)
	if !ok {
		text = "" // Default to empty text if not provided
	}

	// Create the translation
	translation := cmsstore.NewTranslation()
	
	// Set name as the key
	translation.SetName(key)
	
	// Set content using the locale and text
	content := map[string]string{locale: text}
	if err := translation.SetContent(content); err != nil {
		http.Error(w, fmt.Sprintf(`{"success":false,"error":"Failed to set translation content: %v"}`, err), http.StatusBadRequest)
		return
	}
	
	// Store key and locale in meta for easier retrieval
	if err := translation.SetMeta("key", key); err != nil {
		http.Error(w, fmt.Sprintf(`{"success":false,"error":"Failed to set key in meta: %v"}`, err), http.StatusBadRequest)
		return
	}
	
	if err := translation.SetMeta("locale", locale); err != nil {
		http.Error(w, fmt.Sprintf(`{"success":false,"error":"Failed to set locale in meta: %v"}`, err), http.StatusBadRequest)
		return
	}

	// Set site ID - required field
	siteID, ok := translationData["site_id"].(string)
	if !ok || siteID == "" {
		http.Error(w, `{"success":false,"error":"Site ID is required"}`, http.StatusBadRequest)
		return
	}
	translation.SetSiteID(siteID)

	// Save the translation
	if err := api.store.TranslationCreate(r.Context(), translation); err != nil {
		http.Error(w, fmt.Sprintf(`{"success":false,"error":"Failed to save translation: %v"}`, err), http.StatusInternalServerError)
		return
	}

	// Get content to extract text for the specific locale
	translationContent, err := translation.Content()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"success":false,"error":"Failed to get translation content: %v"}`, err), http.StatusInternalServerError)
		return
	}
	
	// Get metas to extract key and locale
	metas, err := translation.Metas()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"success":false,"error":"Failed to get translation metas: %v"}`, err), http.StatusInternalServerError)
		return
	}
	
	// Return the created translation
	response := map[string]interface{}{
		"success": true,
		"id":      translation.ID(),
		"key":     metas["key"],
		"locale":  metas["locale"],
		"text":    translationContent[metas["locale"]],
		"site_id": translation.SiteID(),
		"name":    translation.Name(),
	}

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"success":false,"error":"Failed to create response: %v"}`, err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

// handleTranslationGet handles HTTP requests to get a translation by ID
func (api *RestAPI) handleTranslationGet(w http.ResponseWriter, r *http.Request, translationID string) {
	// Get the translation from the store
	translation, err := api.store.TranslationFindByID(r.Context(), translationID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"success":false,"error":"Failed to find translation: %v"}`, err), http.StatusInternalServerError)
		return
	}

	if translation == nil {
		http.Error(w, `{"success":false,"error":"Translation not found"}`, http.StatusNotFound)
		return
	}

	// Get content to extract text for the specific locale
	translationContent, err := translation.Content()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"success":false,"error":"Failed to get translation content: %v"}`, err), http.StatusInternalServerError)
		return
	}
	
	// Get metas to extract key and locale
	metas, err := translation.Metas()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"success":false,"error":"Failed to get translation metas: %v"}`, err), http.StatusInternalServerError)
		return
	}
	
	// Return the translation
	response := map[string]interface{}{
		"success": true,
		"id":      translation.ID(),
		"key":     metas["key"],
		"locale":  metas["locale"],
		"text":    translationContent[metas["locale"]],
		"site_id": translation.SiteID(),
		"name":    translation.Name(),
	}

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"success":false,"error":"Failed to create response: %v"}`, err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

// handleTranslationList handles HTTP requests to list all translations
func (api *RestAPI) handleTranslationList(w http.ResponseWriter, r *http.Request) {
	// Get query parameters
	siteID := r.URL.Query().Get("site_id")
	locale := r.URL.Query().Get("locale")
	key := r.URL.Query().Get("key")

	// Create translation query
	query := cmsstore.TranslationQuery()
	if siteID != "" {
		query = query.SetSiteID(siteID)
	}
	
	// For key, we can use the name field since we're storing the key there
	if key != "" {
		query = query.SetNameLike(key)
	}
	
	// Note: We can't directly filter by locale in the query
	// We'll need to filter the results after fetching them

	// Get translations from the store
	translations, err := api.store.TranslationList(r.Context(), query)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"success":false,"error":"Failed to list translations: %v"}`, err), http.StatusInternalServerError)
		return
	}

	// Convert translations to response format
	translationsList := make([]map[string]interface{}, 0, len(translations))
	for _, translation := range translations {
		// Get content and metas for each translation
		translationContent, err := translation.Content()
		if err != nil {
			continue // Skip this translation if we can't get content
		}
		
		metas, err := translation.Metas()
		if err != nil {
			continue // Skip this translation if we can't get metas
		}
		
		// Check if we need to filter by locale
		if locale != "" && metas["locale"] != locale {
			continue // Skip translations that don't match the requested locale
		}
		
		translationData := map[string]interface{}{
			"id":      translation.ID(),
			"key":     metas["key"],
			"locale":  metas["locale"],
			"text":    translationContent[metas["locale"]],
			"site_id": translation.SiteID(),
			"name":    translation.Name(),
		}
		translationsList = append(translationsList, translationData)
	}

	// Return the translations list
	response := map[string]interface{}{
		"success":      true,
		"translations": translationsList,
	}

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"success":false,"error":"Failed to create response: %v"}`, err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

// handleTranslationUpdate handles HTTP requests to update a translation
func (api *RestAPI) handleTranslationUpdate(w http.ResponseWriter, r *http.Request, translationID string) {
	// Get the existing translation
	translation, err := api.store.TranslationFindByID(r.Context(), translationID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"success":false,"error":"Failed to find translation: %v"}`, err), http.StatusInternalServerError)
		return
	}

	if translation == nil {
		http.Error(w, `{"success":false,"error":"Translation not found"}`, http.StatusNotFound)
		return
	}

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"success":false,"error":"Failed to read request body: %v"}`, err), http.StatusBadRequest)
		return
	}

	// Parse the request body
	var updates map[string]interface{}
	if err := json.Unmarshal(body, &updates); err != nil {
		http.Error(w, fmt.Sprintf(`{"success":false,"error":"Failed to parse request body: %v"}`, err), http.StatusBadRequest)
		return
	}

	// Get current content and metas
	currentContent, err := translation.Content()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"success":false,"error":"Failed to get translation content: %v"}`, err), http.StatusInternalServerError)
		return
	}
	
	metas, err := translation.Metas()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"success":false,"error":"Failed to get translation metas: %v"}`, err), http.StatusInternalServerError)
		return
	}
	
	// Get current key and locale from metas (for reference only)
	_ = metas["key"]
	_ = metas["locale"]
	
	// Apply updates
	if key, ok := updates["key"].(string); ok && key != "" {
		// Update key in name and meta
		translation.SetName(key)
		if err := translation.SetMeta("key", key); err != nil {
			http.Error(w, fmt.Sprintf(`{"success":false,"error":"Failed to update key in meta: %v"}`, err), http.StatusBadRequest)
			return
		}
	}
	
	// Handle locale update
	newLocale := ""
	if locale, ok := updates["locale"].(string); ok && locale != "" {
		newLocale = locale
		// Update locale in meta
		if err := translation.SetMeta("locale", locale); err != nil {
			http.Error(w, fmt.Sprintf(`{"success":false,"error":"Failed to update locale in meta: %v"}`, err), http.StatusBadRequest)
			return
		}
	} else {
		// Use current locale if not updated
		newLocale = metas["locale"]
	}
	
	// Handle text update
	if text, ok := updates["text"].(string); ok {
		// Update the content map with the new text for the locale
		updatedContent := currentContent
		updatedContent[newLocale] = text
		
		// Set the updated content
		if err := translation.SetContent(updatedContent); err != nil {
			http.Error(w, fmt.Sprintf(`{"success":false,"error":"Failed to update translation content: %v"}`, err), http.StatusBadRequest)
			return
		}
	}

	// Save the updated translation
	if err := api.store.TranslationUpdate(r.Context(), translation); err != nil {
		http.Error(w, fmt.Sprintf(`{"success":false,"error":"Failed to save translation: %v"}`, err), http.StatusInternalServerError)
		return
	}

	// Get updated content and metas for the response
	updatedContent, err := translation.Content()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"success":false,"error":"Failed to get updated translation content: %v"}`, err), http.StatusInternalServerError)
		return
	}
	
	updatedMetas, err := translation.Metas()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"success":false,"error":"Failed to get updated translation metas: %v"}`, err), http.StatusInternalServerError)
		return
	}
	
	// Return the updated translation
	response := map[string]interface{}{
		"success": true,
		"id":      translation.ID(),
		"key":     updatedMetas["key"],
		"locale":  updatedMetas["locale"],
		"text":    updatedContent[updatedMetas["locale"]],
		"site_id": translation.SiteID(),
		"name":    translation.Name(),
	}

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"success":false,"error":"Failed to create response: %v"}`, err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

// handleTranslationDelete handles HTTP requests to delete a translation
func (api *RestAPI) handleTranslationDelete(w http.ResponseWriter, r *http.Request, translationID string) {
	// Delete the translation
	if err := api.store.TranslationSoftDeleteByID(r.Context(), translationID); err != nil {
		http.Error(w, fmt.Sprintf(`{"success":false,"error":"Failed to delete translation: %v"}`, err), http.StatusInternalServerError)
		return
	}

	// Return success response
	response := map[string]interface{}{
		"success": true,
		"message": "Translation deleted successfully",
	}

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"success":false,"error":"Failed to create response: %v"}`, err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}
