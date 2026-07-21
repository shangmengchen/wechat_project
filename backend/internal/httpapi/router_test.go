package httpapi

import "testing"

func TestNewRouterBuilds(t *testing.T) {
	if NewRouter(nil) == nil {
		t.Fatal("router is nil")
	}
}
