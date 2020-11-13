module github.com/coreos/mantle

go 1.12

require (
	cloud.google.com/go/storage v1.9.0
	github.com/Azure/azure-sdk-for-go v8.1.0-beta+incompatible
	github.com/Azure/go-autorest v9.1.0+incompatible
	github.com/Microsoft/azure-vhd-utils v0.0.0-20161127050200-43293b8d7646
	github.com/ajeddeloh/go-json v0.0.0-20200220154158-5ae607161559 // indirect
	github.com/ajeddeloh/yaml v0.0.0-20170912190910-6b94386aeefd // indirect
	github.com/alecthomas/units v0.0.0-20190924025748-f65c72e2690d // indirect
	github.com/aws/aws-sdk-go v1.35.27
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/coreos/bbolt v1.3.1-coreos.6 // indirect
	github.com/coreos/container-linux-config-transpiler v0.9.0
	github.com/coreos/coreos-cloudinit v1.11.0
	github.com/coreos/etcd v3.3.9+incompatible
	github.com/coreos/go-omaha v0.0.0-20180327202221-e409d983eb60
	github.com/coreos/go-semver v0.3.0
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf // indirect
	github.com/coreos/ignition v0.35.0
	github.com/coreos/ignition/v2 v2.7.0
	github.com/coreos/ioprogress v0.0.0-20151023204047-4637e494fd9b
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f
	github.com/coreos/vcontext v0.0.0-20200225161404-ee043618d38d // indirect
	github.com/coreos/yaml v0.0.0-20141224210557-6b16a5714269 // indirect
	github.com/cpuguy83/go-md2man v1.0.10 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/digitalocean/godo v1.45.0
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/fatih/color v1.10.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/godbus/dbus v0.0.0-20181025153459-66d97aec3384
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/golang/protobuf v1.4.3
	github.com/google/go-cmp v0.5.3 // indirect
	github.com/google/uuid v1.1.2 // indirect
	github.com/gophercloud/gophercloud v0.0.0-20180817041643-185230dfbd12
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.6.2 // indirect
	github.com/hashicorp/go-hclog v0.15.0 // indirect
	github.com/hashicorp/go-retryablehttp v0.6.8 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jonboulle/clockwork v0.2.2 // indirect
	github.com/kballard/go-shellquote v0.0.0-20150810074751-d8ec1a69a250
	github.com/kylelemons/godebug v0.0.0-20150519154555-21cb3784d9bd
	github.com/mailru/easyjson v0.0.0-20190403194419-1ea4449da983 // indirect
	github.com/packethost/packngo v0.2.1-0.20200224173249-1156d996f0d5
	github.com/pborman/uuid v1.2.0
	github.com/philhofer/fwd v1.0.0 // indirect
	github.com/pin/tftp v2.1.0+incompatible
	github.com/pquerna/ffjson v0.0.0-20181028064349-e517b90714f7 // indirect
	github.com/prometheus/client_golang v0.9.2 // indirect
	github.com/prometheus/common v0.0.0-20181218105931-67670fe90761 // indirect
	github.com/prometheus/procfs v0.0.2 // indirect
	github.com/sirupsen/logrus v1.4.1 // indirect
	github.com/soheilhy/cmux v0.1.4 // indirect
	github.com/spf13/cobra v0.0.0-20151124153217-1c44ec8d3f15
	github.com/spf13/pflag v0.0.0-20151218134703-7f60f83a2c81
	github.com/tinylib/msgp v1.1.0 // indirect
	github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5 // indirect
	github.com/ugorji/go v0.0.0-20171019201919-bdcc60b419d1 // indirect
	github.com/ulikunitz/xz v0.5.4
	github.com/vincent-petithory/dataurl v0.0.0-20160330182126-9a301d65acbb
	github.com/vishvananda/netlink v0.0.0-20150217221648-2e9d285a7160
	github.com/vishvananda/netns v0.0.0-20150710222425-604eaf189ee8
	github.com/vmware/govmomi v0.22.2
	github.com/xiang90/probing v0.0.0-20160813154853-07dd2e8dfe18 // indirect
	go4.org v0.0.0-20180809161055-417644f6feb5 // indirect
	golang.org/x/crypto v0.0.0-20191011191535-87dc89f01550
	golang.org/x/net v0.0.0-20200602114024-627f9648deb9
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sys v0.0.0-20201113135734-0a15ea8d9b02
	golang.org/x/text v0.3.3
	google.golang.org/api v0.26.0
	google.golang.org/protobuf v1.25.0 // indirect
)

replace google.golang.org/cloud => cloud.google.com/go v0.0.0-20190220171618-cbb15e60dc6d

replace launchpad.net/gocheck => gopkg.in/check.v1 v1.0.0-20200902074654-038fdea0a05b
