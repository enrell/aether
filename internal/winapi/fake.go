package winapi

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// MemoryRegistry is an in-memory Registry for tests. Key names are compared
// case-insensitively, like the real registry.
type MemoryRegistry struct {
	mu     sync.Mutex
	values map[string]map[string]Value
	// Writes counts Set, Delete and DeleteKey calls.
	Writes int
}

// NewMemoryRegistry returns an empty registry.
func NewMemoryRegistry() *MemoryRegistry {
	return &MemoryRegistry{values: map[string]map[string]Value{}}
}

func registryKey(key string) string { return strings.ToLower(strings.Trim(key, `\`)) }

func (r *MemoryRegistry) Get(key, name string) (Value, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	values, ok := r.values[registryKey(key)]
	if !ok {
		return Value{}, false, nil
	}
	value, ok := values[strings.ToLower(name)]
	return value, ok, nil
}

func (r *MemoryRegistry) Set(key, name string, value Value) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Writes++
	k := registryKey(key)
	if r.values[k] == nil {
		r.values[k] = map[string]Value{}
	}
	if value.Kind == KindBinary {
		value.Binary = append([]byte(nil), value.Binary...)
	}
	r.values[k][strings.ToLower(name)] = value
	return nil
}

func (r *MemoryRegistry) Delete(key, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Writes++
	if values, ok := r.values[registryKey(key)]; ok {
		delete(values, strings.ToLower(name))
	}
	return nil
}

func (r *MemoryRegistry) DeleteKey(key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Writes++
	prefix := registryKey(key)
	for k := range r.values {
		if k == prefix || strings.HasPrefix(k, prefix+`\`) {
			delete(r.values, k)
		}
	}
	return nil
}

// Keys returns the keys that hold at least one value, sorted.
func (r *MemoryRegistry) Keys() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	keys := make([]string, 0, len(r.values))
	for k, v := range r.values {
		if len(v) > 0 {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}

// FakeDesktop records desktop calls for tests.
type FakeDesktop struct {
	mu           sync.Mutex
	WallpaperSet string
	Color        *[3]uint8
	Broadcasts   []string
	// Fail makes every mutating call return this error.
	Fail error
}

func (d *FakeDesktop) Wallpaper() (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.WallpaperSet, nil
}

func (d *FakeDesktop) SetWallpaper(path string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.Fail != nil {
		return d.Fail
	}
	d.WallpaperSet = path
	return nil
}

func (d *FakeDesktop) SetDesktopColor(r, g, b uint8) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.Fail != nil {
		return d.Fail
	}
	d.Color = &[3]uint8{r, g, b}
	return nil
}

func (d *FakeDesktop) BroadcastSettingChange(area string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.Fail != nil {
		return d.Fail
	}
	d.Broadcasts = append(d.Broadcasts, area)
	return nil
}

// FakeSystem returns a System backed by fakes.
func FakeSystem() (System, *MemoryRegistry, *FakeDesktop) {
	reg := NewMemoryRegistry()
	desk := &FakeDesktop{}
	return System{Registry: reg, Desktop: desk}, reg, desk
}

var _ fmt.Stringer = Kind(0)
