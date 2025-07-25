package scanner

import (
	"testing"
)

func TestNewLinkParser(t *testing.T) {
	parser := NewLinkParser()
	if parser == nil {
		t.Error("Expected NewLinkParser to return a non-nil parser")
	}
}

func TestIsLocalURL(t *testing.T) {
	parser := NewLinkParser()

	// For now, this always returns false since it's not implemented
	// This test will be updated when the actual implementation is added
	result := parser.IsLocalURL("images/logo.png")
	if result != false {
		t.Error("Expected IsLocalURL to return false (not implemented yet)")
	}
}

func TestParseFile(t *testing.T) {
	parser := NewLinkParser()

	// For now, this returns empty slice since it's not implemented
	// This test will be updated when the actual implementation is added
	refs, err := parser.ParseFile("test.html")
	if err != nil {
		t.Errorf("Expected no error from ParseFile, got: %v", err)
	}

	if len(refs) != 0 {
		t.Errorf("Expected empty slice from ParseFile (not implemented), got %d references", len(refs))
	}
}
