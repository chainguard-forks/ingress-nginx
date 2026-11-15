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

package controller

import (
	"bytes"
	"os"
	"strings"
	"testing"

	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"k8s.io/ingress-nginx/pkg/apis/ingress"
)

// captureKlog returns what klog writes while f runs.
func captureKlog(t *testing.T, f func()) string {
	t.Helper()
	var buf bytes.Buffer
	klog.LogToStderr(false)
	klog.SetOutput(&buf)
	defer func() {
		klog.SetOutput(os.Stderr)
		klog.LogToStderr(true)
	}()
	f()
	klog.Flush()
	return buf.String()
}

// An ImplementationSpecific path can contain a newline: the API server only
// checks that it starts with "/". The controller's log must show it escaped,
// so that it cannot start a line of its own.
func TestMergeAlternativeBackendsQuotesLogValues(t *testing.T) {
	ing := &ingress.Ingress{
		Ingress: networking.Ingress{
			ObjectMeta: metav1.ObjectMeta{Namespace: "example", Name: "canary"},
			Spec: networking.IngressSpec{
				Rules: []networking.IngressRule{{
					Host: "example.com",
					IngressRuleValue: networking.IngressRuleValue{
						HTTP: &networking.HTTPIngressRuleValue{
							Paths: []networking.HTTPIngressPath{{
								Path:     "/x\nFORGED",
								PathType: &pathTypePrefix,
								Backend: networking.IngressBackend{
									Service: &networking.IngressServiceBackend{
										Name: "http-svc-canary",
										Port: networking.ServiceBackendPort{Number: 80},
									},
								},
							}},
						},
					},
				}},
			},
		},
	}
	upstreams := map[string]*ingress.Backend{
		"example-http-svc-canary-80": {Name: "example-http-svc-canary-80", NoServer: true},
	}
	servers := map[string]*ingress.Server{
		"example.com": {
			Hostname: "example.com",
			Locations: []*ingress.Location{
				// The primary backend of this location does not exist.
				{Path: "/x\nFORGED", PathType: &pathTypePrefix, Backend: "example-missing-80"},
				// This location is served by the alternative backend itself.
				{Path: "/y\nFORGED", PathType: &pathTypePrefix, Backend: "example-http-svc-canary-80"},
			},
		},
	}

	out := captureKlog(t, func() { mergeAlternativeBackends(ing, upstreams, servers) })

	for _, want := range []string{
		`cannot find primary backend "example-missing-80" for location "example.com/x\nFORGED"`,
		`alternative upstream "example-http-svc-canary-80" in Ingress "example/canary" is primary upstream in Other Ingress for location "example.com/y\nFORGED"!`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the log does not contain %s; it is:\n%s", want, out)
		}
	}
	if strings.Contains(out, "\nFORGED") {
		t.Errorf("a newline from a path reached the log unescaped:\n%s", out)
	}
}
