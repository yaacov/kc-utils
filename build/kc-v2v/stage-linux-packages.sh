#!/usr/bin/env bash
# Stage RHEL-family qemu-guest-agent RPMs into /usr/share/kc-packages for
# offline Linux guest-agent install (kc-convert-linux --offline).
#
# Layout:
#   $DEST/rpm/el8/x86_64/qemu-guest-agent-*.rpm
#   $DEST/rpm/el9/x86_64/qemu-guest-agent-*.rpm
#   $DEST/rpm/el10/x86_64/qemu-guest-agent-*.rpm
#
# Pins are CentOS Stream Koji builds (subpackages of qemu-kvm). Fail loud on HTTP errors.
set -euo pipefail

DEST="${1:-/usr/share/kc-packages}"
KOJI="${KOJI_BASE:-https://kojihub.stream.centos.org/kojifiles/packages}"

# el_tag|relative_path_under_kojifiles/packages
PINNED_RPMS=(
  "el8|qemu-kvm/6.2.0/36.module_el8+482+7affe3c5/x86_64/qemu-guest-agent-6.2.0-36.module_el8+482+7affe3c5.x86_64.rpm"
  "el9|qemu-kvm/9.1.0/29.el9/x86_64/qemu-guest-agent-9.1.0-29.el9.x86_64.rpm"
  "el10|qemu-kvm/10.1.0/24.el10/x86_64/qemu-guest-agent-10.1.0-24.el10.x86_64.rpm"
)

download() {
  local url="$1" out="$2"
  echo "Downloading $url -> $out"
  # CentOS Koji returns 403 to default curl UA from some networks.
  curl -fL --retry 3 --retry-delay 2 \
    -A "kc-utils-stage-linux-packages/1.0" \
    -o "$out" "$url"
}

for entry in "${PINNED_RPMS[@]}"; do
  el_tag="${entry%%|*}"
  rel="${entry#*|}"
  filename="$(basename "$rel")"
  dir="${DEST}/rpm/${el_tag}/x86_64"
  mkdir -p "$dir"
  download "${KOJI}/${rel}" "${dir}/${filename}"
  # Sanity: non-empty RPM
  if [[ ! -s "${dir}/${filename}" ]]; then
    echo "error: empty download ${dir}/${filename}" >&2
    exit 1
  fi
done

echo "Staged qemu-guest-agent RPMs under ${DEST}/rpm/el{8,9,10}/x86_64/"
find "${DEST}/rpm" -type f -name 'qemu-guest-agent*.rpm' -print
