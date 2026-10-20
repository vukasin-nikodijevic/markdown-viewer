#!/usr/bin/env bash
# Usage: render-formula.sh <version> <checksums.txt>  (prints the formula)
set -euo pipefail
version=$1 checksums=$2
template="$(dirname "$0")/md-viewer.rb.in"

sha() {
  local s
  s=$(awk -v f="md-viewer_${version}_$1.tar.gz" '$2 == f { print $1 }' "$checksums")
  [[ -n "$s" ]] || { echo "no checksum for $1 in $checksums" >&2; exit 1; }
  echo "$s"
}

sed -e "s/@VERSION@/$version/g" \
    -e "s/@SHA256_DARWIN_ARM64@/$(sha darwin_arm64)/" \
    -e "s/@SHA256_DARWIN_AMD64@/$(sha darwin_amd64)/" \
    -e "s/@SHA256_LINUX_ARM64@/$(sha linux_arm64)/" \
    -e "s/@SHA256_LINUX_AMD64@/$(sha linux_amd64)/" \
    "$template"
