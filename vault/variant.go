package vault

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

const (
	variantPrefix       = "__variant__"
	variantActivePrefix = "__variant_active__"
)

// VariantEntry holds a single variant's data.
type VariantEntry struct {
	Index  int
	Value  []byte
	Active bool
}

func variantKey(name string, idx int) string {
	return fmt.Sprintf("%s%s__%d", variantPrefix, name, idx)
}

func activeKey(name string) string {
	return variantActivePrefix + name
}

// IsInternalKey reports whether key is an internal variant metadata key.
func IsInternalKey(key string) bool {
	return strings.HasPrefix(key, variantPrefix) || strings.HasPrefix(key, variantActivePrefix)
}

// VariantAdd adds a new variant value for name and returns its index.
// If it's the first variant, it also sets it as active.
func (v *Vault) VariantAdd(name string, value []byte) (int, error) {
	indices, err := v.variantIndices(name)
	if err != nil {
		return 0, err
	}
	next := 1
	if len(indices) > 0 {
		next = indices[len(indices)-1] + 1
	}

	if err := v.Set(variantKey(name, next), value); err != nil {
		return 0, err
	}

	// First variant becomes active automatically.
	if len(indices) == 0 {
		if err := v.setActive(name, next, value); err != nil {
			return 0, err
		}
	}

	return next, nil
}

// VariantUse switches the active variant for name to idx.
func (v *Vault) VariantUse(name string, idx int) error {
	val, err := v.Get(variantKey(name, idx))
	if err != nil {
		return fmt.Errorf("variant %d not found for %q", idx, name)
	}
	return v.setActive(name, idx, val)
}

// VariantList returns all variants for name with active flag set.
func (v *Vault) VariantList(name string) ([]VariantEntry, error) {
	indices, err := v.variantIndices(name)
	if err != nil {
		return nil, err
	}

	activeIdx := 0
	if raw, err := v.Get(activeKey(name)); err == nil {
		activeIdx, _ = strconv.Atoi(string(raw))
	}

	entries := make([]VariantEntry, 0, len(indices))
	for _, idx := range indices {
		val, err := v.Get(variantKey(name, idx))
		if err != nil {
			return nil, err
		}
		entries = append(entries, VariantEntry{
			Index:  idx,
			Value:  val,
			Active: idx == activeIdx,
		})
	}
	return entries, nil
}

// VariantRemove deletes variant idx for name.
// If it was active, switches to the lowest remaining variant (or cleans up if none left).
func (v *Vault) VariantRemove(name string, idx int) error {
	key := variantKey(name, idx)
	if err := v.Delete(key); err != nil {
		return fmt.Errorf("variant %d not found for %q", idx, name)
	}

	// Check if removed variant was active.
	activeIdx := 0
	if raw, err := v.Get(activeKey(name)); err == nil {
		activeIdx, _ = strconv.Atoi(string(raw))
	}

	if activeIdx == idx {
		indices, _ := v.variantIndices(name)
		if len(indices) > 0 {
			val, _ := v.Get(variantKey(name, indices[0]))
			_ = v.setActive(name, indices[0], val)
		} else {
			_ = v.Delete(activeKey(name))
			_ = v.Delete(name)
		}
	}
	return nil
}

func (v *Vault) setActive(name string, idx int, val []byte) error {
	if err := v.Set(name, val); err != nil {
		return err
	}
	return v.Set(activeKey(name), []byte(strconv.Itoa(idx)))
}

// variantIndices returns a sorted slice of variant indices for name.
func (v *Vault) variantIndices(name string) ([]int, error) {
	prefix := fmt.Sprintf("%s%s__", variantPrefix, name)
	all, err := v.st.List()
	if err != nil {
		return nil, err
	}
	var indices []int
	for _, k := range all {
		if strings.HasPrefix(k, prefix) {
			suffix := k[len(prefix):]
			if idx, err := strconv.Atoi(suffix); err == nil {
				indices = append(indices, idx)
			}
		}
	}
	sort.Ints(indices)
	return indices, nil
}
