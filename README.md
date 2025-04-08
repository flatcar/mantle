<div style="text-align: center">

[![Flatcar OS](https://img.shields.io/badge/Flatcar-Website-blue?logo=data:image/svg+xml;base64,PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0idXRmLTgiPz4NCjwhLS0gR2VuZXJhdG9yOiBBZG9iZSBJbGx1c3RyYXRvciAyNi4wLjMsIFNWRyBFeHBvcnQgUGx1Zy1JbiAuIFNWRyBWZXJzaW9uOiA2LjAwIEJ1aWxkIDApICAtLT4NCjxzdmcgdmVyc2lvbj0iMS4wIiBpZD0ia2F0bWFuXzEiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyIgeG1sbnM6eGxpbms9Imh0dHA6Ly93d3cudzMub3JnLzE5OTkveGxpbmsiIHg9IjBweCIgeT0iMHB4Ig0KCSB2aWV3Qm94PSIwIDAgODAwIDYwMCIgc3R5bGU9ImVuYWJsZS1iYWNrZ3JvdW5kOm5ldyAwIDAgODAwIDYwMDsiIHhtbDpzcGFjZT0icHJlc2VydmUiPg0KPHN0eWxlIHR5cGU9InRleHQvY3NzIj4NCgkuc3Qwe2ZpbGw6IzA5QkFDODt9DQo8L3N0eWxlPg0KPHBhdGggY2xhc3M9InN0MCIgZD0iTTQ0MCwxODIuOGgtMTUuOXYxNS45SDQ0MFYxODIuOHoiLz4NCjxwYXRoIGNsYXNzPSJzdDAiIGQ9Ik00MDAuNSwzMTcuOWgtMzEuOXYxNS45aDMxLjlWMzE3Ljl6Ii8+DQo8cGF0aCBjbGFzcz0ic3QwIiBkPSJNNTQzLjgsMzE3LjlINTEydjE1LjloMzEuOVYzMTcuOXoiLz4NCjxwYXRoIGNsYXNzPSJzdDAiIGQ9Ik02NTUuMiw0MjAuOXYtOTUuNGgtMTUuOXY5NS40aC0xNS45VjI2MmgtMzEuOVYxMzQuOEgyMDkuNFYyNjJoLTMxLjl2MTU5aC0xNS45di05NS40aC0xNnY5NS40aC0xNS45djMxLjINCgloMzEuOXYxNS44aDQ3Ljh2LTE1LjhoMTUuOXYxNS44SDI3M3YtMTUuOGgyNTQuOHYxNS44aDQ3Ljh2LTE1LjhoMTUuOXYxNS44aDQ3Ljh2LTE1LjhoMzEuOXYtMzEuMkg2NTUuMnogTTQ4Ny44LDE1MWg3OS42djMxLjgNCgloLTIzLjZ2NjMuNkg1MTJ2LTYzLjZoLTI0LjJMNDg3LjgsMTUxTDQ4Ny44LDE1MXogTTIzMywyMTQuNlYxNTFoNjMuN3YyMy41aC0zMS45djE1LjhoMzEuOXYyNC4yaC0zMS45djMxLjhIMjMzVjIxNC42eiBNMzA1LDMxNy45DQoJdjE1LjhoLTQ3Ljh2MzEuOEgzMDV2NDcuN2gtOTUuNVYyODYuMUgzMDVMMzA1LDMxNy45eiBNMzEyLjYsMjQ2LjRWMTUxaDMxLjl2NjMuNmgzMS45djMxLjhMMzEyLjYsMjQ2LjRMMzEyLjYsMjQ2LjRMMzEyLjYsMjQ2LjR6DQoJIE00NDguMywzMTcuOXY5NS40aC00Ny44di00Ny43aC0zMS45djQ3LjdoLTQ3LjhWMzAyaDE1Ljl2LTE1LjhoOTUuNVYzMDJoMTUuOUw0NDguMywzMTcuOXogTTQ0MCwyNDYuNHYtMzEuOGgtMTUuOXYzMS44aC0zMS45DQoJdi03OS41aDE1Ljl2LTE1LjhoNDcuOHYxNS44aDE1Ljl2NzkuNUg0NDB6IE01OTEuNiwzMTcuOXY0Ny43aC0xNS45djE1LjhoMTUuOXYzMS44aC00Ny44di0zMS43SDUyOHYtMTUuOGgtMTUuOXY0Ny43aC00Ny44VjI4Ni4xDQoJaDEyNy4zVjMxNy45eiIvPg0KPC9zdmc+DQo=)](https://www.flatcar.org/)
[![Matrix](https://img.shields.io/badge/Matrix-Chat%20with%20us!-green?logo=matrix)](https://app.element.io/#/room/#flatcar:matrix.org)
[![Slack](https://img.shields.io/badge/Slack-Chat%20with%20us!-4A154B?logo=slack)](https://kubernetes.slack.com/archives/C03GQ8B5XNJ)
[![Twitter Follow](https://img.shields.io/twitter/follow/flatcar?style=social)](https://x.com/flatcar)
[![Mastodon Follow](https://img.shields.io/badge/Mastodon-Follow-6364FF?logo=mastodon)](https://hachyderm.io/@flatcar)
[![Bluesky](https://img.shields.io/badge/Bluesky-Follow-0285FF?logo=bluesky)](https://bsky.app/profile/flatcar.org)

</div>
# Mantle: Gluing Container Linux together

This repository is a collection of utilities for developing Container Linux. Most of the
tools are for uploading, running, and interacting with Container Linux instances running
locally or in a cloud.

## Overview
Mantle is composed of many utilities:
 - `cork` for handling the Container Linux SDK
 - `gangue` for downloading from Google Storage
 - `kola` for launching instances and running tests
 - `kolet` an agent for kola that runs on instances
 - `ore` for interfacing with cloud providers
 - `plume` for releasing Container Linux

All of the utilities support the `help` command to get a full listing of their subcommands
and options.

## Tools

### cork
Cork is a now-deprecated tool that was used to help in working with Container Linux images and the SDK.

Please see [developer guides](https://www.flatcar.org/docs/latest/reference/developer-guides/) to see how to work with Flatcar SDK.

### gangue
Gangue is a tool for downloading and verifying files from Google Storage with authenticated requests.
It is primarily used by the SDK.

#### gangue get
Get a file from Google Storage and verify it using GPG.

### kola
Kola is a framework for testing software integration in Container Linux instances
across multiple platforms. It is primarily designed to operate within
the Container Linux SDK for testing software that has landed in the OS image.
Ideally, all software needed for a test should be included by building
it into the image from the SDK.

Kola supports running tests on multiple platforms, currently QEMU, GCE,
AWS, VMware VSphere, Packet, and OpenStack. In the future systemd-nspawn and other
platforms may be added.
Machines on cloud platforms do not have direct access to the kola so tests may depend on
Internet services such as discovery.etcd.io or quay.io instead.

Kola outputs assorted logs and test data to `_kola_temp` for later
inspection.

Kola is still under heavy development and it is expected that its
interface will continue to change.

By default, kola uses the `qemu` platform with the image
`/mnt/host/source/src/build/images/BOARD/latest/flatcar_production_image.bin`.

#### kola run

##### Getting started with QEMU

The easiest way to get started with `kola` is to run a `qemu` test.

***requirements***:
 - IPv4 forwarding (to provide internet access to the instance): `sudo sysctl -w net.ipv4.ip_forward=1`
 - Stop `firewalld.service` or similar frameworks: `sudo systemctl stop firewalld.service` (for permanent disablement use `sudo systemctl disable --now firewalld.service`)
 - `swtpm`, `dnsmasq`, `go` and `iptables` installed and present in the `$PATH`
 - `qemu-system-x86_64` and / or `qemu-system-aarch64` to respectively tests `amd64` and / or `arm64`

From the pulled sources, `kola` and `kolet` must be compiled:

```shell
git clone https://github.com/flatcar/mantle/
cd mantle
./build kola kolet
```

Alternatively, there is a container image with the required dependencies and the mantle binaries for the latest commit on `flatcar-master`:

```
sudo docker run --privileged --net host -v /dev:/dev --rm -it ghcr.io/flatcar/mantle:git-$(git rev-parse HEAD)
# inside the container you can run "kola …" because it is in the PATH, and "sudo kola" is also not needed
```

Finally, a Flatcar image must be available on the system:
  - from a locally [built](https://www.flatcar.org/docs/latest/reference/developer-guides/sdk-modifying-flatcar/) image
  - from an official [release](https://www.flatcar.org/releases)

###### Run tests for AMD64

Example with the latest `alpha` release:
```shell
wget https://alpha.release.flatcar-linux.net/amd64-usr/current/flatcar_production_qemu_image.img
wget https://alpha.release.flatcar-linux.net/amd64-usr/current/flatcar_production_qemu_image.img.sig
gpg --verify flatcar_production_qemu_image.img.sig

wget https://alpha.release.flatcar-linux.net/amd64-usr/current/flatcar_production_qemu_uefi_efi_code.qcow2
wget https://alpha.release.flatcar-linux.net/amd64-usr/current/flatcar_production_qemu_uefi_efi_code.qcow2.sig
gpg --verify flatcar_production_qemu_uefi_efi_code.qcow2.sig

wget https://alpha.release.flatcar-linux.net/amd64-usr/current/flatcar_production_qemu_uefi_efi_vars.qcow2
wget https://alpha.release.flatcar-linux.net/amd64-usr/current/flatcar_production_qemu_uefi_efi_vars.qcow2.sig
gpg --verify flatcar_production_qemu_uefi_efi_vars.qcow2.sig

sudo ./bin/kola run --board amd64-usr --key ${HOME}/.ssh/id_rsa.pub -k -b cl -p qemu \
    --qemu-firmware flatcar_production_qemu_uefi_efi_code.qcow2 \
    --qemu-ovmf-vars flatcar_production_qemu_uefi_efi_vars.qcow2 \
    --qemu-image flatcar_production_qemu_image.img \
    cl.locksmith.cluster
```

###### Run tests for ARM64
Example with the latest `alpha` release:
```shell
wget https://alpha.release.flatcar-linux.net/arm64-usr/current/flatcar_production_qemu_uefi_image.img
wget https://alpha.release.flatcar-linux.net/arm64-usr/current/flatcar_production_qemu_uefi_image.img.sig
gpg --verify flatcar_production_qemu_uefi_image.img.sig

wget https://alpha.release.flatcar-linux.net/arm64-usr/current/flatcar_production_qemu_uefi_efi_code.qcow2
wget https://alpha.release.flatcar-linux.net/arm64-usr/current/flatcar_production_qemu_uefi_efi_code.qcow2.sig
gpg --verify flatcar_production_qemu_uefi_efi_code.qcow2.sig

sudo ./bin/kola run --board arm64-usr --key ${HOME}/.ssh/id_rsa.pub -k -b cl -p qemu \
    --qemu-firmware flatcar_production_qemu_uefi_efi_code.qcow2 \
    --qemu-image flatcar_production_qemu_uefi_image.img \
    cl.etcd-member.discovery
```

_Note for both architectures_:
- `sudo` is required because we need to create some `iptables` rules to provide QEMU Internet access
- using `--remove=false -d`, it's possible to keep the instances running (even after the test) and identify the PID of QEMU instances to SSH into (running processes must be killed once the action done)
- using `--key`, it's possible to SSH into the created instances - PID identification of the `qemu` instance is required:
```shell
ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ProxyCommand="sudo nsenter -n -t <PID of the QEMU instance> nc %h %p" -p 22 core@<IP of the QEMU instance>
```
- using `--qemu-vnc 0`, it's possible to setup a VNC server. Similar to SSH you need to identify the PID of the `qemu` instance to setup a proxy:
```
mkfifo reply
nc -kl 12800 < reply | sudo nsenter -t "${QEMUPID}" -n nc localhost 5900 > reply
rm reply
```
Now, you can access the VNC session on localhost:12800 using a VNC client.

##### Advanced usage with Equinix Metal

The advantage of Kola is to be able to run tests for every supported provider without duplicating testing code. Running tests on Equinix Metal is a bit different from other providers as it boots from PXE.

The test is split into two phases:
* the initial PXE booting with the Flatcar installation
* the actual Flatcar booting with the userdata defined in the test

For this two phases, Kola needs to temporary store two files:
* an Ignition config
* an iPXE configuration

It's possible to use Google Cloud Storage or a regular webserver to host these two files. For the webserver, it needs two requirements:
* a webserver accessible from the Equinix Metal instance
* a remote access to this webserver

For example, the following command:
```
BASENAME="test-em"
BOARD="amd64-usr"
EQUINIXMETAL_KEY="1234"
CHANNEL="alpha"
RELEASE="3255.0.0"1
EQUINIXMETAL_PROJECT="5678"
./bin/kola run --basename=${BASENAME} --board=${BOARD} \
  --equinixmetal-api-key=${PACKET_KEY} \
  --equinixmetal-image-url=https://bucket.release.flatcar-linux.net/flatcar-jenkins/${CHANNEL}/boards/${BOARD}/${RELEASE}/flatcar_production_packet_image.bin.bz2 \
  --equinixmetal-installer-image-base-url=https://bucket.release.flatcar-linux.net/flatcar-jenkins/${CHANNEL}/boards/${BOARD}/${RELEASE} \
  --equinixmetal-project=${EQUINIXMETAL_PROJECT} \
  --equinixmetal-storage-url="ssh+https://my-server" \
  --equinixmetal-remote-document-root="/var/www" \
  --equinixmetal-remote-user="core" \
  --equinixmetal-remote-ssh-private-key-path="./id_rsa" \
  --platform=equinixmetal \
  ${TEST_NAME}
```

will upload the temporary files into "/var/www" using "ssh -i ./id_rsa core@my-server" and the iPXE, Ignition URL will be served at: "https://my-server/mantle-12345.{ipxe,ign}"

#### kola list
The list command lists all of the available tests.

#### kola spawn
The spawn command launches Container Linux instances.

#### kola mkimage
The mkimage command creates a copy of the input image with its primary console set
to the serial port (/dev/ttyS0). This causes more output to be logged on the console,
which is also logged in `_kola_temp`. This can only be used with QEMU images and must
be used with the `coreos_*_image.bin` image, *not* the `coreos_*_qemu_image.img`.

#### kola bootchart
The bootchart command launches an instance then generates an svg of the boot process
using `systemd-analyze`.

#### kola updatepayload
The updatepayload command launches a Container Linux instance then updates it by
sending an update to its update_engine. The update is the `coreos_*_update.gz` in the
latest build directory.

#### kola subtest parallelization
Subtests can be parallelized by adding `c.H.Parallel()` at the top of the inline function
given to `c.Run`. It is not recommended to utilize the `FailFast` flag in tests that utilize
this functionality as it can have unintended results.

#### kola test namespacing
The top-level namespace of tests should fit into one of the following categories:
1. Groups of tests targeting specific packages/binaries may use that
namespace (ex: `docker.*`)
2. Tests that target multiple supported distributions may use the
`coreos` namespace.
3. Tests that target singular distributions may use the distribution's
namespace.

#### kola test registration
Registering kola tests currently requires that the tests are registered
under the kola package and that the test function itself lives within
the mantle codebase.

Groups of similar tests are registered in an init() function inside the
kola package.  `Register(*Test)` is called per test. A kola `Test`
struct requires a unique name, and a single function that is the entry
point into the test. Additionally, userdata (such as a Container Linux
Config) can be supplied. See the `Test` struct in
[kola/register/register.go](https://github.com/flatcar/mantle/tree/master/kola/register/register.go)
for a complete list of options.

#### kola test writing
A kola test is a go function that is passed a `platform.TestCluster` to
run code against.  Its signature is `func(platform.TestCluster)`
and must be registered and built into the kola binary. 

A `TestCluster` implements the `platform.Cluster` interface and will
give you access to a running cluster of Container Linux machines. A test writer
can interact with these machines through this interface.

To see test examples look under
[kola/tests](https://github.com/flatcar/mantle/tree/master/kola/tests) in the
mantle codebase.

For a quickstart see [kola/README.md](/kola/README.md).

#### kola native code
For some tests, the `Cluster` interface is limited and it is desirable to
run native go code directly on one of the Container Linux machines. This is
currently possible by using the `NativeFuncs` field of a kola `Test`
struct. This like a limited RPC interface.

`NativeFuncs` is used similar to the `Run` field of a registered kola
test. It registers and names functions in nearby packages.  These
functions, unlike the `Run` entry point, must be manually invoked inside
a kola test using a `TestCluster`'s `RunNative` method. The function
itself is then run natively on the specified running Container Linux instances.

For more examples, look at the
[coretest](https://github.com/flatcar/mantle/tree/master/kola/tests/coretest)
suite of tests under kola. These tests were ported into kola and make
heavy use of the native code interface.

#### Manhole
The `platform.Manhole()` function creates an interactive SSH session which can
be used to inspect a machine during a test.

### kolet
kolet is run on kola instances to run native functions in tests. Generally kolet
is not invoked manually.

### ore
Ore provides a low-level interface for each cloud provider. It has commands
related to launching instances on a variety of platforms (gcloud, aws,
azure, esx, and packet) within the latest SDK image. Ore mimics the underlying
api for each cloud provider closely, so the interface for each cloud provider
is different. See each providers `help` command for the available actions.

Note, when uploading to some cloud providers (e.g. gce) the image may need to be packaged
with a different --format (e.g. --format=gce) when running `image_to_vm.sh`

### plume
Plume is the Container Linux release utility. Releases are done in two stages,
each with their own command: pre-release and release. Both of these commands are idempotent.

#### plume pre-release
The pre-release command does as much of the release process as possible without making anything public.
This includes uploading images to cloud providers (except those like gce which don't allow us to upload
images without making them public).

### plume release
Publish a new Container Linux release. This makes the images uploaded by pre-release public and uploads
images that pre-release could not. It copies the release artifacts to public storage buckets and updates
the directory index.

#### plume index
Generate and upload index.html objects to turn a Google Cloud Storage
bucket into a publicly browsable file tree. Useful if you want something
like Apache's directory index for your software download repository.
Plume release handles this as well, so it does not need to be run as part of
the release process.

## Platform Credentials
Each platform reads the credentials it uses from different files. The `aws`, `azure`, `do`, `esx` and `packet`
platforms support selecting from multiple configured credentials, call "profiles". The examples below
are for the "default" profile, but other profiles can be specified in the credentials files and selected
via the `--<platform-name>-profile` flag:
```
kola spawn -p aws --aws-profile other_profile
```

### aws
`aws` reads the `~/.aws/credentials` file used by Amazon's aws command-line tool.
It can be created using the `aws` command:
```
$ aws configure
```
To configure a different profile, use the `--profile` flag
```
$ aws configure --profile other_profile
```

The `~/.aws/credentials` file can also be populated manually:
```
[default]
aws_access_key_id = ACCESS_KEY_ID_HERE
aws_secret_access_key = SECRET_ACCESS_KEY_HERE
```

To install the `aws` command in the SDK, run:
```
sudo emerge --ask awscli
```

### azure
`azure` uses `~/.azure/azureProfile.json`. This can be created using the `az` [command](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli):
```
$ az login`
```
It also requires that the environment variable `AZURE_AUTH_LOCATION` points to a JSON file (this can also be set via the `--azure-auth` parameter). The JSON file will require a service provider active directory account to be created.

Service provider accounts can be created via the `az` command (the output will contain an `appId` field which is used as the `clientId` variable in the `AZURE_AUTH_LOCATION` JSON):
```
az ad sp create-for-rbac
```

The client secret can be created inside of the Azure portal when looking at the service provider account under the `Azure Active Directory` service on the `App registrations` tab.

You can find your subscriptionId & tenantId in the `~/.azure/azureProfile.json` via:
```
cat ~/.azure/azureProfile.json | jq '{subscriptionId: .subscriptions[].id, tenantId: .subscriptions[].tenantId}'
```

The JSON file exported to the variable `AZURE_AUTH_LOCATION` should be generated by hand and have the following contents:
```
{
  "clientId": "<service provider id>", 
  "clientSecret": "<service provider secret>", 
  "subscriptionId": "<subscription id>", 
  "tenantId": "<tenant id>", 
  "activeDirectoryEndpointUrl": "https://login.microsoftonline.com", 
  "resourceManagerEndpointUrl": "https://management.azure.com/", 
  "activeDirectoryGraphResourceId": "https://graph.windows.net/", 
  "sqlManagementEndpointUrl": "https://management.core.windows.net:8443/", 
  "galleryEndpointUrl": "https://gallery.azure.com/", 
  "managementEndpointUrl": "https://management.core.windows.net/"
}

```

### do
`do` uses `~/.config/digitalocean.json`. This can be configured manually:
```
{
    "default": {
        "token": "token goes here"
    }
}
```

### esx
`esx` uses `~/.config/esx.json`. This can be configured manually:
```
{
    "default": {
        "server": "server.address.goes.here",
        "user": "user.goes.here",
        "password": "password.goes.here"
    }
}
```

### gce
`gce` uses the `~/.boto` file. When the `gce` platform is first used, it will print
a link that can be used to log into your account with gce and get a verification code
you can paste in. This will populate the `.boto` file.

See [Google Cloud Platform's Documentation](https://cloud.google.com/storage/docs/boto-gsutil)
for more information about the `.boto` file.

### openstack
`openstack` uses `~/.config/openstack.json`. This can be configured manually:
```
{
    "default": {
        "auth_url": "auth url here",
        "tenant_id": "tenant id here",
        "tenant_name": "tenant name here",
        "username": "username here",
        "password": "password here",
        "user_domain": "domain id here",
        "floating_ip_pool": "floating ip pool here",
        "region_name": "region here"
    }
}
```

`user_domain` is required on some newer versions of OpenStack using Keystone V3 but is optional on older versions. `floating_ip_pool` and `region_name` can be optionally specified here to be used as a default if not specified on the command line.

### packet
`packet` uses `~/.config/packet.json`. This can be configured manually:
```
{
	"default": {
		"api_key": "your api key here",
		"project": "project id here"
	}
}
```

### qemu
`qemu` is run locally and needs no credentials, but does need to be run as root.

### qemu-unpriv
`qemu-unpriv` is run locally and needs no credentials. It has a restricted set of functionality compared to the `qemu` platform, such as:

- Single node only, no machine to machine networking
- DHCP provides no data (forces several tests to be disabled)
- No [Local cluster](platform/local/)
