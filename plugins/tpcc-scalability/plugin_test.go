package main

import (
	"testing"
)

func TestMetadata(t *testing.T) {
	p := NewPlugin()
	md := p.Metadata()
	if md.Name != "tpcc-scalability" {
		t.Errorf("Metadata.Name = %q, want %q", md.Name, "tpcc-scalability")
	}
	if md.Version != "2.0.0" {
		t.Errorf("Metadata.Version = %q, want %q", md.Version, "2.0.0")
	}
}
