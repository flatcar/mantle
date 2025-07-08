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

## Core Commands

Kola provides several key commands for testing and development:

- **`kola run`** - Execute specific tests on various platforms
- **`kola list`** - List all available tests  
- **`kola spawn`** - Launch interactive instances for debugging

For complete command documentation, options, and examples, see [kola/README.md](kola/README.md).

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
