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
RUN echo "deb http://packages.cloud.google.com/apt cloud-sdk main" > /etc/apt/sources.list.d/google-cloud-sdk.list && curl https://packages.cloud.google.com/apt/doc/apt-key.gpg > /etc/apt/trusted.gpg.d/cloud.google.gpg && apt-get update -y && apt-get install --no-install-recommends -y python && apt-get install -y google-cloud-cli
COPY --from=builder-amd64 /usr/src/mantle/bin-amd64 /usr/local/bin-amd64
COPY --from=builder-amd64 /usr/src/mantle/bin-arm64 /usr/local/bin-arm64
RUN bash -c 'if [ "$(uname -m)" == "x86_64" ]; then rm -rf /usr/local/bin /usr/local/bin-arm64 ; mv /usr/local/bin-amd64 /usr/local/bin ; else rm -rf /usr/local/bin /usr/local/bin-amd64 ; mv /usr/local/bin-arm64 /usr/local/bin ; fi'
RUN ln -s /usr/share/seabios/bios-256k.bin /usr/share/qemu/bios-256k.bin

# For KVM to work, run the resulting container as: docker run --privileged --net host -v /dev:/dev --rm -it TAG
