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
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	ngx_config "k8s.io/ingress-nginx/internal/ingress/controller/config"
)

// streamServicesStore returns the same ConfigMap, Service and EndpointSlices
// for any key.
type streamServicesStore struct {
	fakeIngressStore
	configMap *corev1.ConfigMap
	service   *corev1.Service
	slices    []*discoveryv1.EndpointSlice
}

func (s *streamServicesStore) GetConfigMap(_ string) (*corev1.ConfigMap, error) {
	return s.configMap, nil
}

func (s *streamServicesStore) GetService(_ string) (*corev1.Service, error) {
	return s.service, nil
}

func (s *streamServicesStore) GetServiceEndpointsSlices(_ string) ([]*discoveryv1.EndpointSlice, error) {
	return s.slices, nil
}

func TestGetStreamServicesPortNumbers(t *testing.T) {
	for _, proto := range []corev1.Protocol{corev1.ProtocolTCP, corev1.ProtocolUDP} {
		t.Run(string(proto), func(t *testing.T) {
			httpName, numericName, letterName := "http", "65536", "4294967296a"
			httpPort, numericPort, letterPort := int32(8080), int32(8443), int32(8444)
			ready := true

			s := &streamServicesStore{
				configMap: &corev1.ConfigMap{
					Data: map[string]string{
						"9000": "default/app:80",
						"9001": "default/app:http",
						// 2^32 + 80, which matched port 80 when it was converted to int32.
						"9002": "default/app:4294967376",
						// A number is never looked up as a port name, even when it is not a valid port.
						"9003": "default/app:65536",
						"9004": "default/app:0",
						// Not a number, although its digits alone overflow int32.
						"9005": "default/app:4294967296a",
					},
				},
				service: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "default"},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{Name: httpName, Port: 80, TargetPort: intstr.FromInt32(httpPort), Protocol: proto},
							{Name: numericName, Port: 443, TargetPort: intstr.FromInt32(numericPort), Protocol: proto},
							{Name: letterName, Port: 444, TargetPort: intstr.FromInt32(letterPort), Protocol: proto},
						},
					},
				},
				slices: []*discoveryv1.EndpointSlice{{
					Ports: []discoveryv1.EndpointPort{
						{Name: &httpName, Port: &httpPort, Protocol: &proto},
						{Name: &numericName, Port: &numericPort, Protocol: &proto},
						{Name: &letterName, Port: &letterPort, Protocol: &proto},
					},
					Endpoints: []discoveryv1.Endpoint{{
						Addresses:  []string{"10.0.0.1"},
						Conditions: discoveryv1.EndpointConditions{Ready: &ready},
					}},
				}},
			}
			n := &NGINXController{
				store: s,
				cfg: &Configuration{
					ListenPorts: &ngx_config.ListenPorts{HTTP: 80, HTTPS: 443, Health: 10254, Default: 8181, SSLProxy: 442},
				},
			}

			var got []int
			for _, svc := range n.getStreamServices("default/stream-services", proto) {
				got = append(got, svc.Port)
			}

			if want := []int{9000, 9001, 9005}; !reflect.DeepEqual(got, want) {
				t.Errorf("getStreamServices returned services on ports %v; want %v", got, want)
			}
		})
	}
}

func TestExternalNamePorts(t *testing.T) {
	svc := &corev1.Service{
		Spec: corev1.ServiceSpec{
			Type:         corev1.ServiceTypeExternalName,
			ExternalName: "example.com",
			Ports: []corev1.ServicePort{
				{Name: "http", Port: 80, TargetPort: intstr.FromInt32(8080)},
				{Name: "65536", Port: 443, TargetPort: intstr.FromInt32(8443)},
				{Name: "4294967296a", Port: 444, TargetPort: intstr.FromInt32(8444)},
			},
		},
	}

	tests := []struct {
		in         string
		port       int32
		targetPort intstr.IntOrString
	}{
		{"80", 80, intstr.FromInt32(8080)},
		{"http", 80, intstr.FromInt32(8080)},
		// A number that matches no Service port is used as it is.
		{"8000", 8000, intstr.FromInt32(8000)},
		// A name that matches no Service port gives port 0.
		{"https", 0, intstr.FromInt32(0)},
		// 2^32 + 80, which matched port 80 when it was converted to int32.
		{"4294967376", 0, intstr.FromInt32(0)},
		// A number is never looked up as a port name, even when it is not a valid port.
		{"65536", 0, intstr.FromInt32(0)},
		// Not a number, although its digits alone overflow int32.
		{"4294967296a", 444, intstr.FromInt32(8444)},
	}

	for _, tt := range tests {
		got := externalNamePorts(tt.in, svc)
		if got.Port != tt.port || got.TargetPort != tt.targetPort {
			t.Errorf("externalNamePorts(%q) = port %d, target port %s; want port %d, target port %s",
				tt.in, got.Port, got.TargetPort.String(), tt.port, tt.targetPort.String())
		}
	}
}
