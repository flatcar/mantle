#!/bin/bash

target_arch=${1}

# Enforce single python version.
pst=$(emerge --info | grep -o 'PYTHON_SINGLE_TARGET="[^"]*"')
pt=${pst/PYTHON_SINGLE_TARGET/PYTHON_TARGETS}
printf '%s\n' "# Enforce single python version" "${pst}" "${pt}" >>/etc/portage/make.conf

# Make sure we will use binary packages, so we can avoid building the
# whole llvm+clang stack.
mkdir -p /etc/portage/binrepos.conf
arch1=${target_arch}
arch2=${target_arch}
if [[ ${target_arch} = 'amd64' ]]; then arch2='x86-64'; fi
uri="https://distfiles.gentoo.org/releases/${arch1}/binpackages/23.0/${arch2}"
cat >/etc/portage/binrepos.conf/gentoobinhost.conf <<BINHOSTEOF
[gentoo]
priority = 1
sync-uri = ${uri}
location = /var/cache/binhost/gentoo
verify-signature = true
BINHOSTEOF
printf '%s\n' '# Enable binary packages' 'FEATURES="${FEATURES} getbinpkg binpkg-request-signature"' >>/etc/portage/make.conf

# Allow the latest version of dev-util/bcc, even if it's unstable.
mkdir -p /etc/portage/package.accept_keywords
echo 'dev-util/bcc' >/etc/portage/package.accept_keywords/bcc

# Install bcc deps and then bcc itself.
#
# "emerge --getbinpkgonly --onlydeps dev-util/bcc" does not work if
# dev-util/bcc has no binary packages, even if we are not installing
# it.
emerge --pretend --verbose --onlydeps dev-util/bcc | grep -e '^\[' >bcc-deps
sed -i -e 's#^\[[^]]*\][[:space:]]*\([^[:space:]]*\)::.*#=\1#' bcc-deps
emerge --getbinpkgonly --verbose --oneshot $(<bcc-deps) || exit 1
rm -f bcc-deps
emerge --verbose dev-util/bcc || exit 1

# Drop the FEATURES line we added for binpkgs, so we can generate the
# binary packages without dabbling with signatures and verifications.
head -n -2 /etc/portage/make.conf >/etc/portage/make.conf.tmp
mv /etc/portage/make.conf.tmp /etc/portage/make.conf
rm -rf /etc/portage/binrepos.conf

# Generate binpkgs from local system.
quickpkg --include-config y $(ls -d1 /var/db/pkg/*/* | sed -e 's#/var/db/pkg/#=#') || exit 1

# Create rootfs, avoid installing perl and docs into the rootfs,
# install bcc and do some cleanups.
mkdir -p /bccrootfs
emerge --nodeps --root=/bccrootfs --usepkgonly sys-apps/baselayout || exit 1
mkdir -p /etc/portage/profile/package.provided
emerge --pretend --quiet --nodeps dev-lang/perl | grep -o 'dev-lang/perl-[0-9.]*' >/etc/portage/profile/package.provided/perl
echo 'INSTALL_MASK="${INSTALL_MASK} /usr/share/locale /usr/share/doc /usr/share/man /usr/share/i18n"' >>/etc/portage/make.conf
emerge --root=/bccrootfs --usepkgonly --verbose --tree dev-util/bcc || exit 1
rm -rf /bccrootfs/var/db/pkg /bccrootfs/var/cache/edb
