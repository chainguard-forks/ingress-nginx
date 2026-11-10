#!/usr/bin/perl

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

# Checks the harness itself: Test::Nginx starts the binary from this
# directory and gets a response, the binary is the nginx version that
# rootfs/build.sh downloads, and the patch stack is applied exactly when
# the build says so (00_drop-alias-root.patch removes the root directive).

###############################################################################

use warnings;
use strict;

use File::Basename qw/ dirname /;
use Test::More;

use Test::Nginx;

###############################################################################

my $t = Test::Nginx->new()->has(qw/http rewrite/)->plan(3)
	->write_file_expand('nginx.conf', <<'EOF');

%%TEST_GLOBALS%%

daemon off;

events {
}

http {
    %%TEST_GLOBALS_HTTP%%

    server {
        listen       127.0.0.1:8080;
        server_name  localhost;

        location / {
            return 200 "harness ok\n";
        }
    }
}

EOF

$t->run();

###############################################################################

my %info;
my $info_file = dirname(dirname($Test::Nginx::NGINX)) . '/build-info';
if (open my $fh, '<', $info_file) {
	%info = map { chomp; split /=/, $_, 2 } <$fh>;
	close $fh;
}

like(http_get('/'), qr/harness ok/, 'request served');

SKIP: {
skip "no $info_file", 2 unless %info;

like(`$Test::Nginx::NGINX -v 2>&1`, qr!nginx/\Q$info{version}\E$!m,
	"binary is nginx $info{version}");

$t->write_file('probe.conf', "events {}\nhttp { root html; }\n");
my $d = $t->testdir();
my $out = `$Test::Nginx::NGINX -t -q -p $d/ -c probe.conf -e probe.log -g 'pid $d/probe.pid;' 2>&1`;
my $flavor = $? == 0 ? 'pristine'
	: $out =~ /unknown directive "root"/ ? 'patched' : "nginx -t failed: $out";

is($flavor, $info{flavor}, "patch stack applied for the $info{flavor} build");

}

###############################################################################
