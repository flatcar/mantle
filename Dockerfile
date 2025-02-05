# Explicitly using an docker.io/amd64/ image to avoid binary translation (assuming the build host is amd64)
#
# golang:1.23-bookworm is based on debian:bookworm, this is important to ensure we have libc compatibility for the copied binary

FROM --platform=linux/amd64 docker.io/amd64/golang:1.23-bookworm as builder-amd64
# We use dynamic linking when possible to reduce compile time and binary size
ENV CGO_ENABLED=1
COPY . /usr/src/mantle
# Build both here because variable builder names (to avoid caching and reusing the wrong one) are only supported with buildkit
RUN bash -c 'cd /usr/src/mantle && ./build ; mv bin bin-amd64 ; CGO_ENABLED=0 GOARCH=arm64 ./build ; mv bin bin-arm64'

# See comment above about golang:1.23-bookworm why debian:bookworm is set here
FROM docker.io/library/debian:bookworm
RUN apt-get update && apt-get upgrade -y

# Run apt-get in the loop - recently installing python packages likes to fail…
RUN for i in $(seq 1 5); do if apt-get --fix-broken install --no-install-recommends -y apt-transport-https awscli azure-cli ca-certificates curl dns-root-data dnsmasq git gnupg2 iptables jq lbzip2 nftables ovmf python-is-python3 python3 qemu-efi-aarch64 qemu-system-aarch64 qemu-system-x86 qemu-utils seabios sqlite3 sudo swtpm; then echo "succeeded at ${i}"; break; elif test $i -eq 5; then echo "fail at ${i}!"; exit 1; fi; done

# from https://cloud.google.com/storage/docs/gsutil_install#deb
RUN echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] http://packages.cloud.google.com/apt cloud-sdk main" | tee -a /etc/apt/sources.list.d/google-cloud-sdk.list && curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | gpg --dearmor -o /usr/share/keyrings/cloud.google.gpg && apt-get update -y
RUN for i in $(seq 1 5); do if apt-get install google-cloud-cli -y; then echo "succeeded at ${i}"; break; elif test $i -eq 5; then echo "fail at ${i}!"; exit 1; fi; done
COPY --from=builder-amd64 /usr/src/mantle/bin-amd64 /usr/local/bin-amd64
COPY --from=builder-amd64 /usr/src/mantle/bin-arm64 /usr/local/bin-arm64
RUN bash -c 'if [ "$(uname -m)" == "x86_64" ]; then rm -rf /usr/local/bin /usr/local/bin-arm64 ; mv /usr/local/bin-amd64 /usr/local/bin ; else rm -rf /usr/local/bin /usr/local/bin-amd64 ; mv /usr/local/bin-arm64 /usr/local/bin ; fi'
RUN ln -s /usr/share/seabios/bios-256k.bin /usr/share/qemu/bios-256k.bin

# For KVM to work, run the resulting container as: docker run --privileged --net host -v /dev:/dev --rm -it TAG
