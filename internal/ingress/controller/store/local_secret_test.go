/*
Copyright 2017 The Kubernetes Authors.

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

package store

import "testing"

func TestSSLCertTracker(t *testing.T) {
	tracker := NewSSLCertTracker()

	items := len(tracker.List())
	if items != 0 {
		t.Errorf("expected 0 items in the store but %v returned", items)
	}

	tracker.Add("key", "value")
	items = len(tracker.List())
	if items != 1 {
		t.Errorf("expected 1 item in the store but %v returned", items)
	}

	item, exists := tracker.Get("key")
	if !exists || item == nil {
		t.Errorf("expected an item from the store but none returned")
	}
}

func TestSSLCertTrackerByKeyQuotesKey(t *testing.T) {
	_, err := NewSSLCertTracker().ByKey("default/tls\nFORGED")
	if err == nil {
		t.Fatal("expected an error for a key that is not in the store")
	}
	if want := `local SSL certificate "default/tls\nFORGED" was not found`; err.Error() != want {
		t.Errorf("ByKey error = %q; want %q", err.Error(), want)
	}
}
