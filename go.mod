module github.com/flatcar-linux/mantle

go 1.16

require (
	cloud.google.com/go v0.34.0
	github.com/Azure/azure-sdk-for-go v56.2.0+incompatible
	github.com/Azure/go-autorest/autorest/adal v0.9.14 // indirect
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.8
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/Microsoft/azure-vhd-utils v0.0.0-20210818134022-97083698b75f
	github.com/ajeddeloh/yaml v0.0.0-20170912190910-6b94386aeefd // indirect
	github.com/alecthomas/units v0.0.0-20151022065526-2efee857e7cf // indirect
	github.com/aws/aws-sdk-go v1.19.11
	github.com/beorn7/perks v1.0.0 // indirect
	github.com/coreos/bbolt v1.3.1-coreos.6 // indirect
	github.com/coreos/container-linux-config-transpiler v0.8.0
	github.com/coreos/coreos-cloudinit v1.11.0
	github.com/coreos/etcd v3.3.9+incompatible
	github.com/coreos/go-iptables v0.5.0
	github.com/coreos/go-omaha v0.0.0-20170526203809-f8acb2d7b76c
	github.com/coreos/go-semver v0.3.0
	github.com/coreos/ignition v0.31.0
	github.com/coreos/ignition/v2 v2.0.1
	github.com/coreos/ioprogress v0.0.0-20151023204047-4637e494fd9b
	github.com/coreos/pkg v0.0.0-20161026222926-447b7ec906e5
	github.com/coreos/yaml v0.0.0-20141224210557-6b16a5714269 // indirect
	github.com/cpuguy83/go-md2man v1.0.4 // indirect
	github.com/dgrijalva/jwt-go v0.0.0-00010101000000-000000000000 // indirect
	github.com/digitalocean/godo v1.45.0
	github.com/form3tech-oss/jwt-go v3.2.5+incompatible // indirect
	github.com/godbus/dbus v0.0.0-20181025153459-66d97aec3384
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.0.0-20210803135101-2ebb50f957d6 // indirect
	github.com/golang/groupcache v0.0.0-20190129154638-5b532d6fd5ef // indirect
	github.com/golang/protobuf v1.3.5
	github.com/google/btree v1.0.0 // indirect
	github.com/google/martian v2.1.0+incompatible // indirect
	github.com/googleapis/gax-go v1.0.3 // indirect
	github.com/gophercloud/gophercloud v0.0.0-20180817041643-185230dfbd12
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.6.2 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jonboulle/clockwork v0.1.0 // indirect
	github.com/kballard/go-shellquote v0.0.0-20150810074751-d8ec1a69a250
	github.com/kylelemons/godebug v0.0.0-20150519154555-21cb3784d9bd
	github.com/mailru/easyjson v0.0.0-20190403194419-1ea4449da983 // indirect
	github.com/packethost/packngo v0.2.1-0.20200224173249-1156d996f0d5
	github.com/pborman/uuid v1.2.0
	github.com/philhofer/fwd v1.0.0 // indirect
	github.com/pin/tftp v2.1.0+incompatible
	github.com/pquerna/ffjson v0.0.0-20181028064349-e517b90714f7 // indirect
	github.com/prometheus/client_golang v0.9.2 // indirect
	github.com/prometheus/client_model v0.0.0-20190129233127-fd36f4220a90 // indirect
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
	github.com/vincent-petithory/dataurl v0.0.0-20160330182126-9a301d65acbb
	github.com/vishvananda/netlink v1.1.1-0.20210330154013-f5de75959ad5
	github.com/vishvananda/netns v0.0.0-20210104183010-2eb08e3e575f
	github.com/vmware/govmomi v0.22.2
	github.com/xiang90/probing v0.0.0-20160813154853-07dd2e8dfe18 // indirect
	go4.org v0.0.0-20180809161055-417644f6feb5 // indirect
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	golang.org/x/net v0.0.0-20210505214959-0714010a04ed
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sys v0.0.0-20210615035016-665e8c7367d1
	golang.org/x/text v0.3.6
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	google.golang.org/api v0.1.0
	google.golang.org/genproto v0.0.0-20190611190212-a7e196e89fd3 // indirect
	google.golang.org/grpc v1.19.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

replace github.com/Microsoft/azure-vhd-utils => github.com/kinvolk/azure-vhd-utils v0.0.0-20210818134022-97083698b75f

replace google.golang.org/cloud => cloud.google.com/go v0.0.0-20190220171618-cbb15e60dc6d

replace launchpad.net/gocheck => gopkg.in/check.v1 v1.0.0-20200902074654-038fdea0a05b

replace github.com/dgrijalva/jwt-go => github.com/golang-jwt/jwt/v4 v4.0.0
