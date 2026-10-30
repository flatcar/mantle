package platform

import (
	"errors"
	"sync/atomic"
)

// ErrProvisioningAborted is returned by platforms once the consecutive
// provisioning failure limit has been reached.
var ErrProvisioningAborted = errors.New("provisioning aborted: platform is unavailable")

var (
	// MaxProvisionFailures is the number of consecutive provider failures
	// (capacity, quota, API errors) after which further provisioning is
	// refused. 0 disables the check. Only platforms that call
	// RecordProvisioning take part.
	MaxProvisionFailures int

	provisionFailures atomic.Int32
	provisionAborted  atomic.Bool
)

// ResetProvisioning clears the failure streak and abort latch for a new run.
func ResetProvisioning() {
	provisionFailures.Store(0)
	provisionAborted.Store(false)
}

// ProvisioningAborted reports whether the failure limit has been reached.
// Once set it stays set for the rest of the run.
func ProvisioningAborted() bool {
	return provisionAborted.Load()
}

// RecordProvisioning tracks the outcome of a provider call that creates
// infrastructure (networks, servers, IPs). Guest boot or configuration
// failures must not be reported here.
func RecordProvisioning(err error) {
	if MaxProvisionFailures <= 0 {
		return
	}
	if err == nil {
		provisionFailures.Store(0)
		return
	}
	if int(provisionFailures.Add(1)) >= MaxProvisionFailures {
		provisionAborted.Store(true)
	}
}
