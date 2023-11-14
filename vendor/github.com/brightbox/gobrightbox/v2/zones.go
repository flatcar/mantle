package brightbox

// Zone represents a Zone
// https://api.gb1.brightbox.com/1.0/#zone
type Zone struct {
	ResourceRef
	ID     string
	Handle string
}
