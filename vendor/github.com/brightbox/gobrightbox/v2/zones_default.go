// Code generated by go generate; DO NOT EDIT.

package brightbox

import "context"
import "path"

import "fmt"

const (
	// zoneAPIPath returns the relative URL path to the Zone endpoint
	zoneAPIPath = "zones"
)

// Zones returns the collection view for Zone
func (c *Client) Zones(ctx context.Context) ([]Zone, error) {
	return apiGetCollection[[]Zone](ctx, c, zoneAPIPath)
}

// Zone retrieves a detailed view of one resource
func (c *Client) Zone(ctx context.Context, identifier string) (*Zone, error) {
	return apiGet[Zone](ctx, c, path.Join(zoneAPIPath, identifier))
}

// Zone retrieves a detailed view of one resource using a handle
func (c *Client) ZoneByHandle(ctx context.Context, handle string) (*Zone, error) {
	collection, err := c.Zones(ctx)
	if err != nil {
		return nil, err
	}
	for _, instance := range collection {
		if instance.Handle == handle {
			return &instance, nil
		}
	}
	return nil, fmt.Errorf("Resource with handle '%s' doesn't exist", handle)
}
