package main

import (
	"runtime"
	"testing"

	"github.com/wailsapp/wails/v2/pkg/menu"
)

// TestBuildMenuIncludesDarwinRoles guards the macOS Command-C fix: the native
// App and Edit menus must be present on darwin (the Edit menu is what gives
// macOS a Cmd-C key equivalent). On other platforms the roles must be absent,
// because a role item renders as an empty submenu on Windows.
func TestBuildMenuIncludesDarwinRoles(t *testing.T) {
	m := buildMenu(NewApp())

	hasApp, hasEdit := false, false
	for _, item := range m.Items {
		switch item.Role {
		case menu.AppMenuRole:
			hasApp = true
		case menu.EditMenuRole:
			hasEdit = true
		}
	}

	if runtime.GOOS == "darwin" {
		if !hasApp || !hasEdit {
			t.Fatalf("darwin menu must include App and Edit roles; got app=%v edit=%v", hasApp, hasEdit)
		}
		if len(m.Items) == 0 || m.Items[0].Role != menu.AppMenuRole {
			t.Fatalf("darwin App menu must be the first menu item")
		}
		return
	}

	if hasApp || hasEdit {
		t.Fatalf("non-darwin menu must not contain role items; got app=%v edit=%v", hasApp, hasEdit)
	}
}
