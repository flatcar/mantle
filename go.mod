module github.com/flatcar-linux/mantle

go 1.17

require (
	cloud.google.com/go/storage v1.9.0
	github.com/Azure/azure-sdk-for-go v56.2.0+incompatible
	github.com/Azure/go-autorest/autorest/adal v0.9.14 // indirect
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.8
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/Microsoft/azure-vhd-utils v0.0.0-20210818134022-97083698b75f
	github.com/aws/aws-sdk-go v1.42.41
	github.com/coreos/coreos-cloudinit v1.11.0
	github.com/coreos/go-iptables v0.5.0
	github.com/coreos/go-omaha v0.0.0-20170526203809-f8acb2d7b76c
	github.com/coreos/go-semver v0.3.0
	github.com/coreos/ioprogress v0.0.0-20151023204047-4637e494fd9b
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f
	github.com/coreos/yaml v0.0.0-20141224210557-6b16a5714269 // indirect
	github.com/digitalocean/godo v1.45.0
	github.com/flatcar-linux/container-linux-config-transpiler v0.9.3-0.20220208152502-6e8303479682
	github.com/flatcar-linux/ignition v0.36.2-0.20220221101037-de4e6cc9bbba
	github.com/flatcar-linux/ignition/v2 v2.2.1-0.20220311122140-cb95c51122f5
	github.com/form3tech-oss/jwt-go v3.2.5+incompatible // indirect
	github.com/godbus/dbus v0.0.0-20181025153459-66d97aec3384
	github.com/golang/protobuf v1.5.2
	github.com/gophercloud/gophercloud v0.0.0-20180817041643-185230dfbd12
	github.com/kballard/go-shellquote v0.0.0-20150810074751-d8ec1a69a250
	github.com/kylelemons/godebug v0.0.0-20150519154555-21cb3784d9bd
	github.com/packethost/packngo v0.21.0
	github.com/pborman/uuid v1.2.0
	github.com/pin/tftp v2.1.0+incompatible
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.6-0.20210604193023-d5e0c0615ace
	github.com/stretchr/testify v1.7.0
	github.com/ulikunitz/xz v0.5.10
	github.com/vincent-petithory/dataurl v1.0.0
	github.com/vishvananda/netlink v1.1.1-0.20210330154013-f5de75959ad5
	github.com/vishvananda/netns v0.0.0-20210104183010-2eb08e3e575f
	github.com/vmware/govmomi v0.22.2
	go.etcd.io/etcd/client/pkg/v3 v3.5.2
	go.etcd.io/etcd/server/v3 v3.5.2
	go.uber.org/zap v1.17.0
	golang.org/x/crypto v0.0.0-20211202192323-5770296d904e
	golang.org/x/net v0.0.0-20220114011407-0dd24b26b47d
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sys v0.0.0-20210615035016-665e8c7367d1
	golang.org/x/text v0.3.7
	google.golang.org/api v0.26.0
)

replace github.com/Microsoft/azure-vhd-utils => github.com/kinvolk/azure-vhd-utils v0.0.0-20210818134022-97083698b75f

replace google.golang.org/cloud => cloud.google.com/go v0.0.0-20190220171618-cbb15e60dc6d

replace launchpad.net/gocheck => gopkg.in/check.v1 v1.0.0-20200902074654-038fdea0a05b

replace github.com/dgrijalva/jwt-go => github.com/golang-jwt/jwt/v4 v4.0.0
