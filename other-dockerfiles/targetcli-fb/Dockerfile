# For this image to work /sys, /etc, /run/dbus and /usr/lib/modules
# need to be forwarded, also needs --privileged and --network host.

FROM docker.io/library/debian:bookworm-slim
RUN apt-get update && apt-get upgrade --assume-yes && apt-get install --assume-yes --no-install-recommends kmod targetcli-fb
