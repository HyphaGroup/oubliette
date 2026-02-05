package config

// CredentialRegistry holds all credentials
type CredentialRegistry struct {
	Factory   FactoryCredentials  `json:"factory"`
	GitHub    GitHubCredentials   `json:"github"`
	Providers ProviderCredentials `json:"providers"`
}

// FactoryCredentials holds Factory API key credentials
type FactoryCredentials struct {
	Credentials map[string]FactoryCredential `json:"credentials"`
	Default     string                       `json:"default"`
}

// FactoryCredential is a single Factory API key
type FactoryCredential struct {
	APIKey      string `json:"api_key"`
	Description string `json:"description"`
}

// GitHubCredentials holds GitHub token credentials
type GitHubCredentials struct {
	Credentials map[string]GitHubCredential `json:"credentials"`
	Default     string                      `json:"default"`
}

// GitHubCredential is a single GitHub token
type GitHubCredential struct {
	Token       string `json:"token"`
	Description string `json:"description"`
}

// ProviderCredentials holds AI provider API credentials
type ProviderCredentials struct {
	Credentials map[string]ProviderCredential `json:"credentials"`
	Default     string                        `json:"default"`
}

// ProviderCredential is a single provider API key (Anthropic, OpenAI, etc.)
type ProviderCredential struct {
	Provider    string `json:"provider"` // anthropic, openai, google
	APIKey      string `json:"api_key"`
	Description string `json:"description"`
}

// GetFactoryKey returns the Factory API key for a named credential
func (r *CredentialRegistry) GetFactoryKey(name string) (string, bool) {
	if cred, ok := r.Factory.Credentials[name]; ok {
		return cred.APIKey, true
	}
	return "", false
}

// GetDefaultFactoryKey returns the default Factory API key
func (r *CredentialRegistry) GetDefaultFactoryKey() (string, bool) {
	if r.Factory.Default == "" {
		return "", false
	}
	return r.GetFactoryKey(r.Factory.Default)
}

// GetGitHubToken returns the GitHub token for a named credential
func (r *CredentialRegistry) GetGitHubToken(name string) (string, bool) {
	if cred, ok := r.GitHub.Credentials[name]; ok {
		return cred.Token, true
	}
	return "", false
}

// GetDefaultGitHubToken returns the default GitHub token
func (r *CredentialRegistry) GetDefaultGitHubToken() (string, bool) {
	if r.GitHub.Default == "" {
		return "", false
	}
	return r.GetGitHubToken(r.GitHub.Default)
}

// GetProviderCredential returns a provider credential by name
func (r *CredentialRegistry) GetProviderCredential(name string) (*ProviderCredential, bool) {
	if cred, ok := r.Providers.Credentials[name]; ok {
		return &cred, true
	}
	return nil, false
}

// GetDefaultProviderCredential returns the default provider credential
func (r *CredentialRegistry) GetDefaultProviderCredential() (*ProviderCredential, bool) {
	if r.Providers.Default == "" {
		return nil, false
	}
	return r.GetProviderCredential(r.Providers.Default)
}

// HasFactoryCredential checks if a factory credential exists
func (r *CredentialRegistry) HasFactoryCredential(name string) bool {
	_, ok := r.Factory.Credentials[name]
	return ok
}

// HasGitHubCredential checks if a github credential exists
func (r *CredentialRegistry) HasGitHubCredential(name string) bool {
	_, ok := r.GitHub.Credentials[name]
	return ok
}

// HasProviderCredential checks if a provider credential exists
func (r *CredentialRegistry) HasProviderCredential(name string) bool {
	_, ok := r.Providers.Credentials[name]
	return ok
}

// CredentialInfo represents a credential without sensitive data (for API responses)
type CredentialInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsDefault   bool   `json:"is_default,omitempty"`
}

// ProviderCredentialInfo includes provider type
type ProviderCredentialInfo struct {
	Name        string `json:"name"`
	Provider    string `json:"provider"`
	Description string `json:"description"`
	IsDefault   bool   `json:"is_default,omitempty"`
}

// CredentialsList is the response for project_options
type CredentialsList struct {
	Factory   []CredentialInfo         `json:"factory"`
	GitHub    []CredentialInfo         `json:"github"`
	Providers []ProviderCredentialInfo `json:"providers"`
}

// ListCredentials returns all credentials without sensitive data
func (r *CredentialRegistry) ListCredentials() CredentialsList {
	result := CredentialsList{
		Factory:   make([]CredentialInfo, 0, len(r.Factory.Credentials)),
		GitHub:    make([]CredentialInfo, 0, len(r.GitHub.Credentials)),
		Providers: make([]ProviderCredentialInfo, 0, len(r.Providers.Credentials)),
	}

	for name, cred := range r.Factory.Credentials {
		result.Factory = append(result.Factory, CredentialInfo{
			Name:        name,
			Description: cred.Description,
			IsDefault:   name == r.Factory.Default,
		})
	}

	for name, cred := range r.GitHub.Credentials {
		result.GitHub = append(result.GitHub, CredentialInfo{
			Name:        name,
			Description: cred.Description,
			IsDefault:   name == r.GitHub.Default,
		})
	}

	for name, cred := range r.Providers.Credentials {
		result.Providers = append(result.Providers, ProviderCredentialInfo{
			Name:        name,
			Provider:    cred.Provider,
			Description: cred.Description,
			IsDefault:   name == r.Providers.Default,
		})
	}

	return result
}

// ProviderEnvVar returns the environment variable name for a provider
func ProviderEnvVar(provider string) string {
	switch provider {
	case "anthropic":
		return "ANTHROPIC_API_KEY"
	case "openai":
		return "OPENAI_API_KEY"
	case "google":
		return "GOOGLE_API_KEY"
	default:
		return ""
	}
}
