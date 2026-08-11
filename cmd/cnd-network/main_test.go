package main

import "testing"

func TestDefaultVersionIsDevelopmentVersion(t *testing.T) {
	if version != "0.1.0-dev" {
		t.Fatalf("version = %q", version)
	}
}
