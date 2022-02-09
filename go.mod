module github.com/flatcar-linux/mantle

go 1.16

require (
	cloud.google.com/go/storage v1.9.0
	github.com/Azure/azure-sdk-for-go v56.2.0+incompatible
	github.com/Azure/go-autorest/autorest/adal v0.9.14 // indirect
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.8
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/Microsoft/azure-vhd-utils v0.0.0-20210818134022-97083698b75f
	github.com/aws/aws-sdk-go v1.42.41
	github.com/beorn7/perks v1.0.0 // indirect
	github.com/coreos/bbolt v1.3.1-coreos.6 // indirect
	github.com/coreos/coreos-cloudinit v1.11.0
	github.com/coreos/etcd v3.3.9+incompatible
	github.com/coreos/go-iptables v0.5.0
	github.com/coreos/go-omaha v0.0.0-20170526203809-f8acb2d7b76c
	github.com/coreos/go-semver v0.3.0
	github.com/coreos/ioprogress v0.0.0-20151023204047-4637e494fd9b
	github.com/coreos/pkg v0.0.0-20161026222926-447b7ec906e5
	github.com/coreos/yaml v0.0.0-20141224210557-6b16a5714269 // indirect
	github.com/cpuguy83/go-md2man v1.0.4 // indirect
	github.com/dgrijalva/jwt-go v0.0.0-00010101000000-000000000000 // indirect
	github.com/digitalocean/godo v1.45.0
	github.com/flatcar-linux/container-linux-config-transpiler v0.9.3-0.20220208152502-6e8303479682
	github.com/flatcar-linux/ignition v0.36.1
	github.com/flatcar-linux/ignition/v2 v2.2.1-0.20220107090316-32908ec8bade
	github.com/form3tech-oss/jwt-go v3.2.5+incompatible // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/godbus/dbus v0.0.0-20181025153459-66d97aec3384
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.0.0-20210803135101-2ebb50f957d6 // indirect
	github.com/golang/protobuf v1.4.2
	github.com/gophercloud/gophercloud v0.0.0-20180817041643-185230dfbd12
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.6.2 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jonboulle/clockwork v0.1.0 // indirect
	github.com/kballard/go-shellquote v0.0.0-20150810074751-d8ec1a69a250
	github.com/kylelemons/godebug v0.0.0-20150519154555-21cb3784d9bd
	github.com/mailru/easyjson v0.0.0-20190403194419-1ea4449da983 // indirect
	github.com/packethost/packngo v0.21.0
	github.com/pborman/uuid v1.2.0
	github.com/philhofer/fwd v1.0.0 // indirect
	github.com/pin/tftp v2.1.0+incompatible
	github.com/pquerna/ffjson v0.0.0-20181028064349-e517b90714f7 // indirect
	github.com/prometheus/client_golang v0.9.2 // indirect
	github.com/prometheus/common v0.0.0-20181218105931-67670fe90761 // indirect
	github.com/prometheus/procfs v0.0.2 // indirect
	github.com/russross/blackfriday v1.5.2 // indirect
	github.com/sirupsen/logrus v1.4.1 // indirect
	github.com/soheilhy/cmux v0.1.4 // indirect
	github.com/spf13/cobra v0.0.0-20151124153217-1c44ec8d3f15
	github.com/spf13/pflag v0.0.0-20151218134703-7f60f83a2c81
	github.com/stretchr/testify v1.7.0
	github.com/tinylib/msgp v1.1.0 // indirect
	github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5 // indirect
	github.com/ugorji/go v0.0.0-20171019201919-bdcc60b419d1 // indirect
	github.com/ulikunitz/xz v0.5.10
	github.com/vincent-petithory/dataurl v1.0.0
	github.com/vishvananda/netlink v1.1.1-0.20210330154013-f5de75959ad5
	github.com/vishvananda/netns v0.0.0-20210104183010-2eb08e3e575f
	github.com/vmware/govmomi v0.22.2
	github.com/xiang90/probing v0.0.0-20160813154853-07dd2e8dfe18 // indirect
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
