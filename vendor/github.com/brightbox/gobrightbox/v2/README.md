# Brightbox Go Library

[![Go Reference](https://pkg.go.dev/badge/github.com/brightbox/gobrightbox.svg)](https://pkg.go.dev/github.com/brightbox/gobrightbox)

`gobrightbox` is a [Brightbox](https://www.brightbox.com) [API](https://api.gb1.brightbox.com/1.0/)
client library written in [Go](http://golang.org/).

Documentation is available at [pkg.go.dev](https://pkg.go.dev/github.com/brightbox/gobrightbox)

The following are the instructions for version 1 of the GO library.

[version 2 instructions](https://pkg.go.dev/github.com/brightbox/gobrightbox/v2) are in the v2 directory.

## Install
```sh
go get github.com/brightbox/gobrightbox@vX.Y.Z
```

where X.Y.Z is the [version](https://github.com/brightbox/gobrightbox/releases) you need.

## Authentication

This client does not itself handle authentication. Instead, use the standard
[OAuth2](https://godoc.org/golang.org/x/oauth2) golang library to
[authenticate](https://api.gb1.brightbox.com/1.0/#authentication) and create
tokens.

## Supported Objects

* Full [Server](https://api.gb1.brightbox.com/1.0/#server) support
* Full [Server Group](https://api.gb1.brightbox.com/1.0/#server_group) support
* Full [CloudIP](https://api.gb1.brightbox.com/1.0/#cloud_ip) support
* Full [Firewall Policy](https://api.gb1.brightbox.com/1.0/#firewall_policy) support
* Full [Load Balancer](https://api.gb1.brightbox.com/1.0/#load_balancer) support
* Full [Cloud SQL](https://api.gb1.brightbox.com/1.0/#database_server) support
* Full [Api Client](https://api.gb1.brightbox.com/1.0/#api_client) support
* Basic [Image](https://api.gb1.brightbox.com/1.0/#image) support
* Basic event stream support

## Help

If you need help using this library, drop an email to support at brightbox dot com.

## Licence

This code is released under an MIT License.

Copyright (c) 2015-2022 Brightbox Systems Ltd.
