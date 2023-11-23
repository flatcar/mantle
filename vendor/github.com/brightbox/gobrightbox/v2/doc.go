/*
Package brightbox provides a client to access the Brightbox API

A [Client] is created by passing a context and a config to the [Connect] function

    // Setup connection to API
    client, err := brightbox.Connect(ctx, conf)

There are two types of Config available.
A [clientcredentials.Config] authenticates using an [API Client] and is specific to a particular Brightbox account.
A [passwordcredentials.Config] authenticates using a [Username] and can be used with any authorised Brightbox account.

[API Client]: https://www.brightbox.com/docs/reference/authentication/#api-client-authentication
[Username]: https://www.brightbox.com/docs/reference/authentication/#user-authentication

*/
package brightbox
