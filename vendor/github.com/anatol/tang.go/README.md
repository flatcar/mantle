# Tang.go

Tang.go pure-go library that implement server side of ECMR key exchange. It functionally similar
to [Tang project](https://github.com/latchset/tang).

The library also provides a convenient way to perform ECMR exchange with a specific key

## Usage

```go
package main

import "github.com/anatol/tang.go"

func main() {
	// Start Tang service
	srv := tang.NewServer()
	keySet, _ := tang.ReadKeysFromDir("/var/db/tang")
	srv.Keys = keySet
	srv.Addr = ":0"
	_ = srv.ListenAndServe()
}
```

Or you can operate with keyset directly and do you own server-side exchange manually:
```go
package main

import (
	"github.com/anatol/tang.go"
	"github.com/lestrrat-go/jwx/jwk"
)

func main() {
	ks := tang.NewKeySet()
	priv, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	key, err := jwk.New(priv)
	key.Set(jwk.KeyOpsKey, []jwk.KeyOperation{jwk.KeyOpDeriveKey})
	key.Set(jwk.AlgorithmKey, "ECMR")
	ks.AppendKey(key, true)

	privRec, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	pubRec := privRec.Public()
	keyRec, err := jwk.New(pubRec)
	keyRec.Set(jwk.AlgorithmKey, "ECMR")
	finalKey, err := ks.RecoverKey("$THP_OF_THE_GENERATED_KEY", keyRec)

	var finalKeyPub ecdsa.PublicKey
	finalKey.Raw(&finalKeyPub)
	// finalKeyPub.X and finalKeyPub.Y are going to be your derived values
}
```

## Acknowledgments

This project has been inspired by:

* [tang](https://github.com/latchset/tang)

Important contributions to this project are done by: