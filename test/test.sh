#!/bin/bash

# Copyright 2018 The Kubernetes Authors.
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


if [ -n "$DEBUG" ]; then
	set -x
fi

set -o errexit
set -o nounset
set -o pipefail

mkdir -p /tmp/nginx
if [ -z "${PKG}" ]; then
  echo "PKG must be set"
  exit 1
fi

# The tag kustomize_disable_go_plugin_support builds kustomize without its
# loader for Go plugins. cmd/plugin/commands/ingresses gets that loader, and
# with it Go's "plugin" package, through k8s.io/cli-runtime; nothing here uses
# it. With cgo on, "plugin" makes Go link that test program with gcc, and on
# linux/arm64 Go 1.26 then always asks gcc for the gold linker, which the test
# image does not have ("collect2: fatal error: cannot find 'ld'"). The tag can
# go when go.mod moves to Go 1.27, which asks for gold only where GNU ld is
# too old.
go test -v -tags kustomize_disable_go_plugin_support \
  $(go list "${PKG}/..." | grep -v vendor | grep -v '/test/e2e' | grep -v images | grep -v "docs/examples")
