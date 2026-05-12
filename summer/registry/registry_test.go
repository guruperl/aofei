package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/genelet/winter/genelet"
)

func TestRegistryCoversComponentModules(t *testing.T) {
	registered := make(map[string]bool, len(Entries))
	for _, entry := range Entries {
		if entry.Name == "" || entry.NewModel == nil || entry.NewStorage == nil || entry.NewFilter == nil {
			t.Fatalf("invalid registry entry: %#v", entry)
		}
		if registered[entry.Name] {
			t.Fatalf("duplicate registry entry %q", entry.Name)
		}
		registered[entry.Name] = true
	}

	files, err := filepath.Glob("../*/component.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		module := filepath.Base(filepath.Dir(file))
		if !registered[module] {
			t.Fatalf("component module %q is missing from registry", module)
		}
	}
	for module := range registered {
		component := filepath.Join("..", module, "component.json")
		if _, err := os.Stat(component); err != nil {
			if strings.Contains(err.Error(), "no such file") {
				t.Fatalf("registry module %q has no component.json", module)
			}
			t.Fatal(err)
		}
		if _, err := genelet.LoadComponent(component); err != nil {
			t.Fatalf("registry module %q component did not validate: %v", module, err)
		}
	}
}
