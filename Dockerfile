# Explicitly using an docker.io/amd64/ image to avoid binary translation (assuming the build host is amd64)
# golang:1.19 is based on debian:11, this is important to ensure we have libc compatibility for the copied binary
FROM --platform=linux/amd64 docker.io/amd64/golang:1.19 as builder-amd64
# We use dynamic linking when possible to reduce compile time and binary size
ENV CGO_ENABLED=1
COPY . /usr/src/mantle
# Build both here because variable builder names (to avoid caching and reusing the wrong one) are only supported with buildkit
RUN bash -c 'cd /usr/src/mantle && ./build ; mv bin bin-amd64 ; CGO_ENABLED=0 GOARCH=arm64 ./build ; mv bin bin-arm64'

# See comment above about golang:1.19 why debian:11 is set here
FROM docker.io/library/debian:11
RUN apt-get update && apt-get upgrade -y && apt-get install --no-install-recommends -y qemu-utils qemu-system-x86 qemu-system-aarch64 qemu-efi-aarch64 seabios ovmf lbzip2 sudo dnsmasq gnupg2 git curl iptables nftables dns-root-data ca-certificates sqlite3 jq awscli azure-cli
# from https://cloud.google.com/storage/docs/gsutil_install#deb
RUN echo "deb http://packages.cloud.google.com/apt cloud-sdk main" > /etc/apt/sources.list.d/google-cloud-sdk.list && { curl -fsSL https://packages.cloud.google.com/apt/doc/apt-key.gpg || { echo -n 'xsBNBGKItdQBCADWmKTNZEYWgXy73FvKFY5fRro4tGNa4Be4TZW3wZpct9Cj8EjykU7S9EPoJ3EdKpxFltHRu7QbDi6LWSNA4XxwnudQrYGxnxx6Ru1KBHFxHhLfWsvFcGMwit/znpxtIt9UzqCm2YTEW5NUnzQ4rXYqVQK2FLG4weYJ5bKwkY+ZsnRJpzxdHGJ0pBiqwkMT8bfQdJymUBown+SeuQ2HEqfjVMsIRe0dweD2PHWeWo9fTXsz1Q5abiGckyOVyoN9//DgSvLUocUcZsrWvYPaN+o8lXTO3GYFGNVsx069rxarkeCjOpiQOWrQmywXISQudcusSgmmgfsRZYW7FDBy5MQrABEBAAHNUVJhcHR1cmUgQXV0b21hdGljIFNpZ25pbmcgS2V5IChjbG91ZC1yYXB0dXJlLXNpZ25pbmcta2V5LTIwMjItMDMtMDctMDhfMDFfMDEucHViKcLAYgQTAQgAFgUCYoi11AkQtT3IDRPt7wUCGwMCGQEAAMGoCAB8QBNIIN3Q2D3aahrfkb6axd55zOwR0tnriuJRoPHoNuorOpCv9aWMMvQACNWkxsvJxEF8OUbzhSYjAR534RDigjTetjK2i2wKLz/kJjZbuF4ZXMynCm40eVm1XZqU63U9XR2RxmXppyNpMqQO9LrzGEnNJuh23icaZY6no12axymxcle/+SCmda8oDAfa0iyA2iyg/eU05buZv54MC6RB13QtS+8vOrKDGr7RYp/VYvQzYWm+ck6DvlaVX6VB51BkLl23SQknyZIJBVPm8ttU65EyrrgG1jLLHFXDUqJ/RpNKq+PCzWiyt4uy3AfXK89RczLu3uxiD0CQI0T31u/IzsBNBGKItdQBCADIMMJdRcg0Phv7+CrZz3xRE8Fbz8AN+YCLigQeH0B9lijxkjAFr+thB0IrOu7ruwNY+mvdP6dAewUur+pJaIjEe+4s8JBEFb4BxJfBBPuEbGSxbi4OPEJuwT53TMJMEs7+gIxCCmwioTggTBp6JzDsT/cdBeyWCusCQwDWpqoYCoUWJLrUQ6dOlI7s6p+iIUNIamtyBCwb4izs27HdEpX8gvO9rEdtcb7399HyO3oD4gHgcuFiuZTpvWHdn9WYwPGM6npJNG7crtLnctTR0cP9KutSPNzpySeAniHx8L9ebdD9tNPCWC+OtOcGRrcBeEznkYh1C4kzdP1ORm5upnknABEBAAHCwF8EGAEIABMFAmKItdQJELU9yA0T7e8FAhsMAABJmAgAhRPk/dFj71bU/UTXrkEkZZzE9JzUgan/ttyRrV6QbFZABByf4pYjBj+yLKw3280//JWurKox2uzEq1hdXPedRHICRuh1Fjd00otaQ+wGF3kY74zlWivB6Wp6tnL9STQ1oVYBUv7HhSHoJ5shELyedxxHxurUgFAD+pbFXIiK8cnAHfXTJMcrmPpC+YWEC/DeqIyEcNPkzRhtRSuERXcq1n+KJvMUAKMD/tezwvujzBaaSWapmdnGmtRjjL7IxUeGamVWOwLQbUr+34MwzdeJdcL8fav5LA8Uk0ulyeXdwiAK8FKQsixI+xZvz7HUs8ln4pZwGw/TpvO9cMkHogtgzQ==' | base64 -d ; } ; } > /etc/apt/trusted.gpg.d/cloud.google.gpg && apt-get update -y && apt-get install --no-install-recommends -y python && apt-get install -y google-cloud-cli
COPY --from=builder-amd64 /usr/src/mantle/bin-amd64 /usr/local/bin-amd64
COPY --from=builder-amd64 /usr/src/mantle/bin-arm64 /usr/local/bin-arm64
RUN bash -c 'if [ "$(uname -m)" == "x86_64" ]; then rm -rf /usr/local/bin /usr/local/bin-arm64 ; mv /usr/local/bin-amd64 /usr/local/bin ; else rm -rf /usr/local/bin /usr/local/bin-amd64 ; mv /usr/local/bin-arm64 /usr/local/bin ; fi'
RUN ln -s /usr/share/seabios/bios-256k.bin /usr/share/qemu/bios-256k.bin

# For KVM to work, run the resulting container as: docker run --privileged --net host -v /dev:/dev --rm -it TAG
