package dto

// GeocodingProviderFieldResponse describes one field in a provider's
// structured configuration. Secret values are returned as "***" and may be
// sent back unchanged to preserve the stored value.
type GeocodingProviderFieldResponse struct {
	Key      string `json:"key" example:"api_key"`
	Type     string `json:"type" example:"password"`
	Required bool   `json:"required" example:"true"`
	Secret   bool   `json:"secret" example:"true"`
}

type GeocodingProviderResponse struct {
	ID                   string                           `json:"id" example:"geoapify"`
	Name                 string                           `json:"name" example:"Geoapify"`
	Configured           bool                             `json:"configured" example:"true"`
	HasStoredConfig      bool                             `json:"has_stored_config" example:"true"`
	SupportsAutocomplete bool                             `json:"supports_autocomplete" example:"true"`
	Notice               string                           `json:"notice" example:"public_demo"`
	Fields               []GeocodingProviderFieldResponse `json:"fields"`
	Config               map[string]string                `json:"config"`
	Attribution          []AddressAttribution             `json:"attribution"`
}

type GeocodingAdminResponse struct {
	ActiveProvider string                      `json:"active_provider" example:"geoapify"`
	Precision      string                      `json:"precision" example:"exact"`
	Providers      []GeocodingProviderResponse `json:"providers"`
}

type UpdateGeocodingSettingsRequest struct {
	ActiveProvider string `json:"active_provider" example:"geoapify"`
	Precision      string `json:"precision" example:"exact" validate:"required,oneof=exact locality"`
}

type UpdateGeocodingProviderConfigRequest struct {
	Config map[string]string `json:"config" validate:"required"`
}
