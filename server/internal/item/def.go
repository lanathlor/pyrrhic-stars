package item

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ItemsDir returns the items directory path.
// Checks CODEX_ITEMS_DIR env var first, then falls back to ../shared/items/.
func ItemsDir() string {
	dir := os.Getenv("CODEX_ITEMS_DIR")
	if dir == "" {
		dir = filepath.Join("..", "shared", "items")
	}
	return dir
}

// ItemDef is a data-driven item template loaded from YAML.
type ItemDef struct {
	ID        string     `yaml:"id"`
	Name      string     `yaml:"name"`
	Slot      SlotID     `yaml:"slot"`
	StatLines []StatLine `yaml:"stat_lines"`
}

// DefRegistry maps item definition IDs to their definitions.
// Populated at startup by LoadItems.
var DefRegistry = map[string]*ItemDef{}

// itemsFile is the top-level YAML schema: a list of item definitions.
type itemsFile struct {
	Items []ItemDef `yaml:"items"`
}

// LoadItems reads all .yaml files from dir and populates DefRegistry.
func LoadItems(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("LoadItems: read dir %q: %w", dir, err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return fmt.Errorf("LoadItems: read %q: %w", e.Name(), err)
		}
		var f itemsFile
		if err := yaml.Unmarshal(data, &f); err != nil {
			return fmt.Errorf("LoadItems: parse %q: %w", e.Name(), err)
		}
		for i := range f.Items {
			def := &f.Items[i]
			if def.ID == "" {
				return fmt.Errorf("LoadItems: %q: item at index %d missing id", e.Name(), i)
			}
			DefRegistry[def.ID] = def
		}
	}
	return nil
}
