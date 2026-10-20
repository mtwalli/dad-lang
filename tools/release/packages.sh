#!/bin/sh
# Builds Linux packages from the release binaries:
#   dist/dad_<version>_amd64.deb, dist/dad_<version>_arm64.deb
#   dist/dad-<version>-1.x86_64.rpm, dist/dad-<version>-1.aarch64.rpm (when rpmbuild is installed)
#
#   tools/release/packages.sh 0.2.0
#
# It expects build/dad_linux_amd64/dad and build/dad_linux_arm64/dad, as the release
# workflow builds them. Each package installs /usr/bin/dad, the Arabic name /usr/bin/ضاد,
# and the README and licence under /usr/share/doc/dad.
set -eu

version="${1:?usage: packages.sh <version>}"
maintainer="${DAD_MAINTAINER:-mtwalli <mtwalli@users.noreply.github.com>}"
homepage="https://github.com/mtwalli/dad-lang"
summary="Arabic programming language for web apps"
description="Ḍād (ضاد) is a programming language with Arabic keywords for building
whole web apps: interface, server and data in one language. This package
installs the dad command: run, check, format, serve and build programs."

mkdir -p dist
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT INT TERM

deb() {
  arch="$1"
  bin="build/dad_linux_${arch}/dad"
  [ -f "$bin" ] || { echo "missing $bin" >&2; exit 1; }
  root="$work/deb-$arch"
  mkdir -p "$root/DEBIAN" "$root/usr/bin" "$root/usr/share/doc/dad"
  install -m 0755 "$bin" "$root/usr/bin/dad"
  ln -s dad "$root/usr/bin/ضاد"
  install -m 0644 README.md "$root/usr/share/doc/dad/README.md"
  install -m 0644 LICENSE "$root/usr/share/doc/dad/copyright"
  size="$(du -sk "$root/usr" | cut -f 1)"
  {
    echo "Package: dad"
    echo "Version: $version"
    echo "Architecture: $arch"
    echo "Maintainer: $maintainer"
    echo "Installed-Size: $size"
    echo "Section: devel"
    echo "Priority: optional"
    echo "Homepage: $homepage"
    echo "Description: $summary"
    printf '%s\n' "$description" | sed 's/^/ /'
  } > "$root/DEBIAN/control"
  dpkg-deb --root-owner-group --build "$root" "dist/dad_${version}_${arch}.deb" >/dev/null
  echo "dist/dad_${version}_${arch}.deb"
}

rpm() {
  arch="$1"     # Go's name
  rpmarch="$2"  # rpm's name
  bin="build/dad_linux_${arch}/dad"
  top="$work/rpm-$arch"
  mkdir -p "$top/SPECS" "$top/SOURCES"
  cp "$bin" README.md LICENSE "$top/SOURCES/"
  cat > "$top/SPECS/dad.spec" <<EOF
%global debug_package %{nil}
%global __strip /bin/true
%define _build_id_links none

Name:           dad
Version:        $version
Release:        1
Summary:        $summary
License:        MIT
URL:            $homepage
Packager:       $maintainer

%description
$description

%install
mkdir -p %{buildroot}%{_bindir} %{buildroot}%{_docdir}/dad %{buildroot}%{_licensedir}/dad
install -m 0755 %{_sourcedir}/dad %{buildroot}%{_bindir}/dad
ln -s dad %{buildroot}%{_bindir}/ضاد
install -m 0644 %{_sourcedir}/README.md %{buildroot}%{_docdir}/dad/README.md
install -m 0644 %{_sourcedir}/LICENSE %{buildroot}%{_licensedir}/dad/LICENSE

%files
%{_bindir}/dad
%{_bindir}/ضاد
%doc %{_docdir}/dad/README.md
%license %{_licensedir}/dad/LICENSE
EOF
  rpmbuild --quiet -bb --target "$rpmarch" --define "_topdir $top" "$top/SPECS/dad.spec"
  cp "$top/RPMS/$rpmarch/dad-${version}-1.${rpmarch}.rpm" dist/
  echo "dist/dad-${version}-1.${rpmarch}.rpm"
}

deb amd64
deb arm64
if command -v rpmbuild >/dev/null 2>&1; then
  rpm amd64 x86_64
  rpm arm64 aarch64
else
  echo "rpmbuild is not installed: no .rpm packages" >&2
fi
