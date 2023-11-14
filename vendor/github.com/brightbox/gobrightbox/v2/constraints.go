package brightbox

// CreateDated is a constraint type that selects all Brightbox objects
// with a creation date
type CreateDated interface {
	// The Unix time in seconds the API object was created
	CreatedAtUnix() int64
}
