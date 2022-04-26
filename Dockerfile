# golang:1.17 is based on debian:11, this is important to ensure we have libc compatibility for the copied binary

FROM docker.io/library/golang:1.17 as builder
ENV CGO_ENABLED=1
COPY . /usr/src/mantle
RUN cd /usr/src/mantle && ./build

FROM docker.io/library/debian:11
RUN apt-get update && apt-get upgrade -y && apt-get install --no-install-recommends -y qemu-utils qemu-system-x86 qemu-system-aarch64 qemu-efi-aarch64 seabios ovmf lbzip2 sudo dnsmasq gnupg2 git curl iptables nftables dns-root-data ca-certificates sqlite3
# from https://cloud.google.com/storage/docs/gsutil_install#deb
RUN echo "deb http://packages.cloud.google.com/apt cloud-sdk main" > /etc/apt/sources.list.d/google-cloud-sdk.list && curl https://packages.cloud.google.com/apt/doc/apt-key.gpg > /etc/apt/trusted.gpg.d/cloud.google.gpg && apt-get update -y && apt-get install --no-install-recommends -y python && apt-get install -y google-cloud-cli
COPY --from=builder /usr/src/mantle/bin /usr/local/bin
RUN ln -s /usr/share/seabios/bios-256k.bin /usr/share/qemu/bios-256k.bin

# For KVM to work, run the resulting container as: docker run --privileged --net host -v /dev:/dev --rm -it TAG
