package config

// ModelDefinition represents a model configuration
type ModelDefinition struct {
	Model           string            `json:"model"`
	DisplayName     string            `json:"displayName"`
	BaseURL         string            `json:"baseUrl"`
	APIKey          string            `json:"apiKey"`
	MaxOutputTokens int               `json:"maxOutputTokens"`
	Provider        string            `json:"provider"`
	ExtraHeaders    map[string]string `json:"extraHeaders,omitempty"`
}

// ModelRegistry holds model configurations keyed by shorthand name
type ModelRegistry struct {
	Models map[string]ModelDefinition `json:"models"`
}

// ModelInfo represents model information without sensitive data (for API responses)
type ModelInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Provider    string `json:"provider"`
}

// GetModel returns a model definition by shorthand name
func (r *ModelRegistry) GetModel(name string) (ModelDefinition, bool) {
	model, ok := r.Models[name]
	return model, ok
}

// HasModel checks if a model exists in the registry
func (r *ModelRegistry) HasModel(name string) bool {
	_, ok := r.Models[name]
	return ok
}

// ListModels returns model info for all models (without API keys)
func (r *ModelRegistry) ListModels() []ModelInfo {
	var models []ModelInfo
	for name, def := range r.Models {
		models = append(models, ModelInfo{
			Name:        name,
			DisplayName: def.DisplayName,
			Provider:    def.Provider,
		})
	}
	return models
}

// ResolveModel resolves a model shorthand name to the full model ID.
// If the name is already a full model ID (not in registry), returns it unchanged.
func (r *ModelRegistry) ResolveModel(name string) string {
	if model, ok := r.Models[name]; ok {
		return model.Model
	}
	return name
}
