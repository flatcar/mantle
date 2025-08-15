package stackit

import (
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"k8s.io/utils/ptr"
)

func createNullableString(s string) *iaas.NullableString {
	n := iaas.NullableString{}
	n.Set(ptr.To(s))
	return &n
}
