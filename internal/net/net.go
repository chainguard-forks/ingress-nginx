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
	"fmt"
	_net "net"
	"os"
	"strconv"
	"strings"
)

// ParsePort parses s as a TCP or UDP port number. isNumber reports whether s
// is a decimal integer: an optional "+" or "-" followed by one or more ASCII
// digits. A string that is not, such as a port name, gives false. valid
// reports whether that integer is also a port number, from 1 to 65535. port
// is the number when valid is true, and 0 otherwise.
func ParsePort(s string) (port int32, isNumber, valid bool) {
	digits := s
	if digits != "" && (digits[0] == '+' || digits[0] == '-') {
		digits = digits[1:]
	}
	// Check the whole string first: strconv reports a range error as soon as
	// the digits overflow, before it sees a later character that is not a
	// digit, as in "4294967296a".
	if digits == "" || strings.Trim(digits, "0123456789") != "" {
		return 0, false, false
	}

	// s is an integer, so the only possible error is a range error.
	p, err := strconv.ParseInt(s, 10, 32)
	if err != nil || p < 1 || p > 65535 {
		return 0, true, false
	}
	return int32(p), true, true
}

// IsIPV6 checks if the input contains a valid IPV6 address
func IsIPV6(ip _net.IP) bool {
	return ip != nil && ip.To4() == nil
}

// IsPortAvailable checks if a TCP port is available or not
func IsPortAvailable(p int) bool {
	ln, err := _net.Listen("tcp", fmt.Sprintf(":%v", p))
	defer func() {
		if ln != nil {
			ln.Close()
		}
	}()
	return err == nil
}

// IsIPv6Enabled checks if IPV6 is enabled or not and we have
// at least one configured in the pod
func IsIPv6Enabled() bool {
	// Skip interface checks if the IPv6 kernel feature is disabled.
	disable, err := os.ReadFile("/proc/sys/net/ipv6/conf/all/disable_ipv6")
	if err != nil {
		return false
	}
	if strings.TrimSpace(string(disable)) == "1" {
		return false
	}

	// Check that there are interfaces with IPv6 enabled.
	ifaces, err := os.Stat("/proc/net/if_inet6")
	if err != nil {
		return false
	}
	if ifaces.IsDir() {
		return false
	}

	// Check IPv6 addresses on interfaces.
	addrs, err := _net.InterfaceAddrs()
	if err != nil {
		return false
	}

	for _, addr := range addrs {
		ip, _, err := _net.ParseCIDR(addr.String())
		if err != nil {
			return false
		}
		if IsIPV6(ip) {
			return true
		}
	}

	return false
}
