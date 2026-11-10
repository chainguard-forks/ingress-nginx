#!/bin/bash

# Copyright 2026 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Builds the nginx binary that the nginx-tests suite runs against.
#
# The nginx version, its sha256, the patch loop and the core configure flags
# are read from rootfs/build.sh, so the test binary follows the base image:
# same source tarball, same patch stack, same core and bundled modules.  It
# differs on purpose in three ways:
#
#   - the third-party modules (Lua, ModSecurity, njs, ...) are left out; the
#     tests exercise nginx itself,
#   - it is built with AddressSanitizer and NGX_DEBUG_PALLOC (every pool
#     allocation becomes its own malloc, so overruns inside pools are seen),
#     and Test::Nginx fails a test file on any sanitizer report in its log,
#   - FLAVOR=pristine skips the patch stack, to compare with stock nginx.

set -o errexit
set -o nounset
set -o pipefail

ROOTFS=${ROOTFS:-/src/rootfs}
PREFIX=${PREFIX:-/opt/nginx}
FLAVOR=${FLAVOR:-patched}
BUILD_SH="${ROOTFS}/build.sh"

case "${FLAVOR}" in
  patched|pristine) ;;
  *) echo "FLAVOR must be patched or pristine, not '${FLAVOR}'" >&2; exit 2 ;;
esac

NGINX_VERSION=$(sed -n 's/^export NGINX_VERSION=//p' "${BUILD_SH}")
NGINX_SHA256=$(awk '/^get_src / { h = $2; getline; if ($0 ~ /nginx\.org\/download\/nginx-/) { print h; exit } }' "${BUILD_SH}")
WITH_FLAGS=
eval "$(sed -n '/^WITH_FLAGS="/,/"$/p' "${BUILD_SH}")"
WITHOUT_FLAGS=$(grep -oE -- '--without-[a-z0-9_]+' "${BUILD_SH}" | tr '\n' ' ')

if [[ -z "${NGINX_VERSION}" || -z "${NGINX_SHA256}" || -z "${WITH_FLAGS}" || -z "${WITHOUT_FLAGS}" ]]; then
  echo "could not read the nginx version, sha256 or configure flags from ${BUILD_SH}" >&2
  exit 3
fi

# same architecture rule as build.sh
if [[ "$(uname -m)" != "aarch64" ]]; then
  WITH_FLAGS+=" --with-file-aio"
fi

BUILD_DIR=$(mktemp -d)
cd "${BUILD_DIR}"

curl -fsSL -o nginx.tar.gz "https://nginx.org/download/nginx-${NGINX_VERSION}.tar.gz"
echo "${NGINX_SHA256}  nginx.tar.gz" | sha256sum -c -
tar -xzf nginx.tar.gz
cd "nginx-${NGINX_VERSION}"

applied=0
if [[ "${FLAVOR}" == "patched" ]]; then
  # same order and -p rule as the patch loop in build.sh
  for PATCH in $(ls "${ROOTFS}/patches"); do
    echo "Patch: ${PATCH}"
    if [[ "${PATCH}" == *.txt ]]; then
      patch -p0 < "${ROOTFS}/patches/${PATCH}"
    else
      patch -p1 < "${ROOTFS}/patches/${PATCH}"
    fi
    applied=$((applied + 1))
  done
fi

# -Wno-error: the patch stack drops -Werror (18_*no_Werror.patch); do the same
# for the pristine build so both flavors compile with the same warnings.
./configure \
  --prefix="${PREFIX}" \
  ${WITH_FLAGS} \
  ${WITHOUT_FLAGS} \
  --with-cc-opt="-g -O1 -fno-omit-frame-pointer -fsanitize=address -DNGX_DEBUG_PALLOC=1 -Wno-error" \
  --with-ld-opt="-fsanitize=address"

make -j"$(nproc)"
make install

printf 'version=%s\nflavor=%s\npatches=%s\n' "${NGINX_VERSION}" "${FLAVOR}" "${applied}" \
  > "${PREFIX}/build-info"
cat "${PREFIX}/build-info"
"${PREFIX}/sbin/nginx" -V

cd /
rm -rf "${BUILD_DIR}"
