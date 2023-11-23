package brightbox

// ResourceRef contains the header fields in every API object
type ResourceRef struct {
	URL          string `json:"url"`
	ResourceType string `json:"resource_type"`
}
