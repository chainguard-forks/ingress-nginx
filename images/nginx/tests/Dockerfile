# Copyright 2026 The Kubernetes Authors. All rights reserved.
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

# Test image for the patched nginx: builds nginx from the version and patch
# stack in ../rootfs with AddressSanitizer (build-nginx.sh), then runs the
# pinned nginx-tests suite and the fork's own tests in t/ (run.sh).
# Build context: images/nginx.  Usage: images/nginx/README.md.

# glibc base: AddressSanitizer does not support musl, so Alpine cannot host it
FROM ubuntu:24.04@sha256:fe81560990fae86dc4743202b2cb4d3df98d0bcf949173cecef15ce1127ef852

# libpcre3: build.sh and the shipped packages link PCRE1, not PCRE2.
# The Perl modules and memcached are what the suite needs to run its SSL,
# HTTP/3, FastCGI and memcached tests instead of skipping them.
RUN apt-get update \
  && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    build-essential \
    ca-certificates \
    curl \
    libcache-memcached-perl \
    libcryptx-perl \
    libfcgi-perl \
    libio-socket-ssl-perl \
    libnet-ssleay-perl \
    libpcre3-dev \
    libssl-dev \
    memcached \
    openssl \
    patch \
    perl \
    zlib1g-dev \
  && rm -rf /var/lib/apt/lists/*

# Check for recent changes: https://github.com/nginx/nginx-tests/compare/16964ee752509eb6271c4e8760e7899274240755...master
ARG NGINX_TESTS_COMMIT=16964ee752509eb6271c4e8760e7899274240755
ARG NGINX_TESTS_SHA256=b8c445ac5f9134b0e6798624e619fd92558bfa9280a2b84c4b42e5c83e922dc8

RUN curl -fsSL -o /tmp/nginx-tests.tar.gz \
    "https://github.com/nginx/nginx-tests/archive/${NGINX_TESTS_COMMIT}.tar.gz" \
  && echo "${NGINX_TESTS_SHA256}  /tmp/nginx-tests.tar.gz" | sha256sum -c - \
  && mkdir -p /opt/nginx-tests \
  && tar -xzf /tmp/nginx-tests.tar.gz -C /opt/nginx-tests --strip-components=1 \
  && rm /tmp/nginx-tests.tar.gz

COPY tests/nginx-tests-no-root.patch /src/tests/
RUN patch -d /opt/nginx-tests -p1 < /src/tests/nginx-tests-no-root.patch

COPY rootfs/build.sh /src/rootfs/build.sh
COPY rootfs/patches /src/rootfs/patches
COPY tests/build-nginx.sh /src/tests/build-nginx.sh

# patched (the default) or pristine (stock nginx, no patch stack)
ARG FLAVOR=patched
RUN /src/tests/build-nginx.sh

COPY tests/run.sh tests/skip.txt /src/tests/
COPY tests/t /opt/fork-tests

RUN useradd --create-home tester
USER tester

ENTRYPOINT ["/src/tests/run.sh"]
