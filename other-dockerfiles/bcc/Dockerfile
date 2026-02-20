FROM gentoo/portage:latest AS portage

FROM gentoo/stage3:latest

# This is to know if we are building for amd64 or arm64.
ARG TARGETARCH

# Copy the entire portage volume in.
COPY --from=portage /var/db/repos/gentoo /var/db/repos/gentoo

# Make sure we will use binary packages, so we can avoid building the
# whole llvm+clang stack. Also build the latest version of
# dev-util/bcc.
RUN <<EOF
mkdir -p /etc/portage/binrepos.conf
arch1=${TARGETARCH}
arch2=${TARGETARCH}
if [[ ${TARGETARCH} = 'amd64' ]]; then arch2='x86-64'; fi
uri="https://distfiles.gentoo.org/releases/${arch1}/binpackages/23.0/${arch2}"
cat >/etc/portage/binrepos.conf/gentoobinhost.conf <<BINHOSTEOF
[gentoo]
priority = 1
sync-uri = ${uri}
location = /var/cache/binhost/gentoo
verify-signature = true
BINHOSTEOF
echo 'FEATURES="${FEATURES} getbinpkg binpkg-request-signature"' >>/etc/portage/make.conf
mkdir -p /etc/portage/package.accept_keywords
echo 'dev-util/bcc' >/etc/portage/package.accept_keywords/bcc
EOF

RUN emerge dev-util/bcc
