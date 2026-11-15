/*
Copyright 2026 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ingress

import (
	"bytes"
	"strings"
	"testing"

	"k8s.io/klog/v2"

	"k8s.io/ingress-nginx/pkg/apis/ingress"
)

// captureKlog returns what klog writes while f runs.
func captureKlog(t *testing.T, f func()) string {
	t.Helper()
	state := klog.CaptureState()
	defer state.Restore()
	var buf bytes.Buffer
	klog.LogToStderr(false)
	klog.SetOutput(&buf)
	f()
	klog.Flush()
	return buf.String()
}

func TestBuildRedirectsQuotesLogValues(t *testing.T) {
	servers := []*ingress.Server{{
		Hostname:          "example.com\nFORGED",
		RedirectFromToWWW: true,
		SSLCert:           &ingress.SSLCert{CN: []string{"other.example"}},
	}}

	out := captureKlog(t, func() { BuildRedirects(servers) })

	want := `the server "www.example.com\nFORGED" has SSL configured but the SSL certificate does not contains a CN for "example.com\nFORGED".`
	if !strings.Contains(out, want) {
		t.Errorf("the log does not contain %s; it is:\n%s", want, out)
	}
	if strings.Contains(out, "\nFORGED") {
		t.Errorf("a newline from a host name reached the log unescaped:\n%s", out)
	}
}
