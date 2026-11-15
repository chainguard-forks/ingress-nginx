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
	"errors"
	"strings"
	"testing"

	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"k8s.io/ingress-nginx/internal/ingress/controller/ingressclass"
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

// checkEscaped reports an error unless out contains each of want, and
// contains no newline followed by "FORGED", the marker the tests put after a
// newline in an Ingress value.
func checkEscaped(t *testing.T, out string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Errorf("the log does not contain %s; it is:\n%s", w, out)
		}
	}
	if strings.Contains(out, "\nFORGED") {
		t.Errorf("a newline from an Ingress value reached the log unescaped:\n%s", out)
	}
}

// noClassStore reports that no Ingress belongs to this controller's class.
type noClassStore struct {
	fakeIngressStore
}

func (noClassStore) GetIngressClass(_ *networking.Ingress, _ *ingressclass.Configuration) (string, error) {
	return "", errors.New("no matching IngressClass")
}

// The admission webhook logs the Ingress it ignores. Its name comes from the
// request, which a caller of the webhook controls.
func TestCheckIngressQuotesLogValues(t *testing.T) {
	ing := &networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "x\nFORGED", Namespace: "default"},
	}

	t.Run("another ingress class", func(t *testing.T) {
		n := &NGINXController{cfg: &Configuration{}, store: &noClassStore{}}
		out := captureKlog(t, func() {
			if err := n.CheckIngress(ing); err != nil {
				t.Errorf("CheckIngress returned %v; want nil", err)
			}
		})
		checkEscaped(t, out, `ignoring ingress "x\nFORGED" in "default"`)
	})

	t.Run("another namespace", func(t *testing.T) {
		n := &NGINXController{cfg: &Configuration{Namespace: "watched"}, store: &fakeIngressStore{}}
		out := captureKlog(t, func() {
			if err := n.CheckIngress(ing); err != nil {
				t.Errorf("CheckIngress returned %v; want nil", err)
			}
		})
		checkEscaped(t, out, `ignoring ingress "x\nFORGED" in namespace "default" different from the namespace watched watched`)
	})
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

	checkEscaped(t, out,
		`cannot find primary backend "example-missing-80" for location "example.com/x\nFORGED"`,
		`alternative upstream "example-http-svc-canary-80" in Ingress "example/canary" is primary upstream in Other Ingress for location "example.com/y\nFORGED"!`,
	)
}
