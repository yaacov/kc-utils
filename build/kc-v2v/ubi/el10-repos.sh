#!/bin/bash
# Enable CentOS Stream 10 + EPEL 10 on a UBI 10 image for packages outside
# the free UBI subset. Intended for Containerfile RUN (root, dnf available).
set -euo pipefail

ARCH="$(uname -m)"
CS_VER="10.0-23.el10"
CS_BASE="https://mirror.stream.centos.org/10-stream/BaseOS/${ARCH}/os/Packages"

dnf install -y --setopt=install_weak_deps=False \
	"${CS_BASE}/centos-gpg-keys-${CS_VER}.noarch.rpm" \
	"${CS_BASE}/centos-stream-repos-${CS_VER}.noarch.rpm"

# CRB is required by some EPEL packages.
dnf config-manager --set-enabled crb

dnf install -y --setopt=install_weak_deps=False \
	https://dl.fedoraproject.org/pub/epel/epel-release-latest-10.noarch.rpm

dnf clean all
