/*
Copyright 2015 The Kubernetes Authors.

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

package net

import (
	"net"
	"testing"
)

func TestIsIPV6(t *testing.T) {
	tests := []struct {
		in     net.IP
		isIPV6 bool
	}{
		{net.ParseIP("2001:4860:4860::8844"), true},
		{net.ParseIP("2001:4860:4860::8888"), true},
		{net.ParseIP("0:0:0:0:0:ffff:c868:8165"), true},
		{net.ParseIP("2001:db8:85a3::8a2e:370:7334"), true},
		{net.ParseIP("::1"), true},
		{net.ParseIP("8.8.8.8"), false},
	}

	for _, test := range tests {
		isIPV6 := IsIPV6(test.in)
		if isIPV6 && !test.isIPV6 {
			t.Fatalf("%v expected %v but returned %v", test.in, test.isIPV6, isIPV6)
		}
	}
}

func TestIsPortAvailable(t *testing.T) {
	if !IsPortAvailable(0) {
		t.Fatal("expected port 0 to be available (random port) but returned false")
	}

	ln, err := net.Listen("tcp", ":0") //nolint:gosec // Ignore the gosec error in testing
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer ln.Close()

	p := ln.Addr().(*net.TCPAddr).Port
	if IsPortAvailable(p) {
		t.Fatalf("expected port %v to not be available", p)
	}
}

func TestParsePort(t *testing.T) {
	tests := []struct {
		in       string
		port     int32
		isNumber bool
		valid    bool
	}{
		{"1", 1, true, true},
		{"80", 80, true, true},
		{"+80", 80, true, true},
		{"080", 80, true, true},
		{"65535", 65535, true, true},
		{"0", 0, true, false},
		{"-1", 0, true, false},
		{"65536", 0, true, false},
		// 2^32 + 80: a conversion from a 64-bit int to int32 would give 80.
		{"4294967376", 0, true, false},
		{"99999999999999999999", 0, true, false},
		{"http", 0, false, false},
		{"8080a", 0, false, false},
		// strconv.ParseInt reports a range error for these before it reaches the letter.
		{"4294967296a", 0, false, false},
		{"99999999999999999999a", 0, false, false},
		{" 80", 0, false, false},
		{"80 ", 0, false, false},
		{"+-1", 0, false, false},
		{"+", 0, false, false},
		{"-", 0, false, false},
		{"", 0, false, false},
	}

	for _, tt := range tests {
		port, isNumber, valid := ParsePort(tt.in)
		if port != tt.port || isNumber != tt.isNumber || valid != tt.valid {
			t.Errorf("ParsePort(%q) = %d, %v, %v; want %d, %v, %v",
				tt.in, port, isNumber, valid, tt.port, tt.isNumber, tt.valid)
		}
	}
}
