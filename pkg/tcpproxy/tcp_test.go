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

package tcpproxy

import (
	"crypto/tls"
	"net"
	"testing"
	"time"

	"pault.ag/go/sniff/parser"
)

// A ClientHello whose SNI extension declares a length (0xffff) larger than the record. Without the recover
// in Handle, pault.ag/go/sniff slices past the buffer and panics; the passthrough goroutine has no other
// guard, so the process would crash.
func craftedClientHello() []byte {
	h := []byte{0x16, 0x03, 0x01, 0x00, 0x40, 0x01, 0x00, 0x00, 0x3c, 0x03, 0x03}
	h = append(h, make([]byte, 32)...)                                        // random
	h = append(h, 0x00)                                                       // session id length 0
	h = append(h, 0x00, 0x02, 0x13, 0x01)                                     // cipher suites
	h = append(h, 0x01, 0x00)                                                 // compression methods
	h = append(h, 0x00, 0x08, 0x00, 0x00, 0xff, 0xff, 0x00, 0x00, 0x00, 0x00) // extensions: SNI, declared length 0xffff
	return h
}

// A panic in Handle's goroutine that is not recovered crashes the test binary (it cannot be caught from
// here), so this test completing at all is the assertion that Handle recovers.
func TestHandleRecoversFromMalformedClientHello(t *testing.T) {
	srv, cli := net.Pipe()
	go func() {
		_, _ = cli.Write(craftedClientHello())
		_ = cli.Close()
	}()
	done := make(chan struct{})
	go func() {
		p := &TCPProxy{ServerList: []*TCPServer{{Hostname: "a.example", IP: "127.0.0.1", Port: 1}}}
		p.Handle(srv) // must return, not panic the process
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Handle did not return for a malformed ClientHello")
	}
}

// Trimming the buffer to the bytes actually read must not break SNI parsing of a real ClientHello.
func TestGetHostnameOnRealClientHello(t *testing.T) {
	srv, cli := net.Pipe()
	go func() { _ = tls.Client(cli, &tls.Config{ServerName: "a.example", InsecureSkipVerify: true}).Handshake() }()
	buf := make([]byte, 16384)
	_ = srv.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := srv.Read(buf)
	if err != nil {
		t.Fatalf("reading the ClientHello: %v", err)
	}
	host, err := parser.GetHostname(buf[:n])
	if err != nil || host != "a.example" {
		t.Fatalf("GetHostname(buf[:%d]) = %q, %v; want \"a.example\", nil", n, host, err)
	}
	_ = srv.Close()
	_ = cli.Close()
}
