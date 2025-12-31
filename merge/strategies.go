package merge

import "fmt"

// concatArrays concatenates two arrays.
func (m *Merger) concatArrays(a, b any) (any, error) {
	aArr, aIsArr := a.([]any)
	bArr, bIsArr := b.([]any)

	if !aIsArr && !bIsArr {
		return nil, fmt.Errorf("concat strategy requires arrays")
	}

	if !bIsArr {
		return aArr, nil
	}
	if !aIsArr {
		return bArr, nil
	}

	result := make([]any, 0, len(bArr)+len(aArr))
	result = append(result, bArr...)
	result = append(result, aArr...)
	return result, nil
}

// concatUniqueArrays concatenates two arrays and removes duplicates.
func (m *Merger) concatUniqueArrays(a, b any) (any, error) {
	concat, err := m.concatArrays(a, b)
	if err != nil {
		return nil, err
	}

	arr, ok := concat.([]any)
	if !ok {
		return concat, nil
	}

	seen := make(map[any]bool)
	result := make([]any, 0, len(arr))
	for _, item := range arr {
		// Only primitive types can be map keys
		if isPrimitive(item) {
			if !seen[item] {
				seen[item] = true
				result = append(result, item)
			}
		} else {
			// For non-primitives, always include (can't easily dedupe)
			result = append(result, item)
		}
	}
	return result, nil
}

// sumNumbers adds two numeric values.
func (m *Merger) sumNumbers(a, b any) (any, error) {
	aNum, aOk := toFloat64(a)
	bNum, bOk := toFloat64(b)

	if !aOk && !bOk {
		return nil, fmt.Errorf("sum strategy requires numbers")
	}
	if !aOk {
		return b, nil
	}
	if !bOk {
		return a, nil
	}

	return aNum + bNum, nil
}

// maxNumber returns the larger of two numeric values.
func (m *Merger) maxNumber(a, b any) (any, error) {
	aNum, aOk := toFloat64(a)
	bNum, bOk := toFloat64(b)

	if !aOk && !bOk {
		return nil, fmt.Errorf("max strategy requires numbers")
	}
	if !aOk {
		return b, nil
	}
	if !bOk {
		return a, nil
	}

	if aNum > bNum {
		return a, nil
	}
	return b, nil
}

// minNumber returns the smaller of two numeric values.
func (m *Merger) minNumber(a, b any) (any, error) {
	aNum, aOk := toFloat64(a)
	bNum, bOk := toFloat64(b)

	if !aOk && !bOk {
		return nil, fmt.Errorf("min strategy requires numbers")
	}
	if !aOk {
		return b, nil
	}
	if !bOk {
		return a, nil
	}

	if aNum < bNum {
		return a, nil
	}
	return b, nil
}

// toFloat64 converts a value to float64 if it's a number.
func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case int32:
		return float64(n), true
	default:
		return 0, false
	}
}

// isPrimitive returns true if the value is a primitive type (can be a map key).
func isPrimitive(v any) bool {
	switch v.(type) {
	case string, int, int64, int32, float64, float32, bool:
		return true
	default:
		return false
	}
}

// mergeByKey merges two arrays of objects by a key field.
// Items with matching keys are merged (or replaced if replaceOnMatch is true);
// items only in A or B are included.
func (m *Merger) mergeByKey(a, b any, keyField string, replaceOnMatch bool, path string) (any, error) {
	aArr, aIsArr := a.([]any)
	bArr, bIsArr := b.([]any)

	if !aIsArr && !bIsArr {
		return nil, fmt.Errorf("mergeByKey strategy requires arrays")
	}

	if !bIsArr || len(bArr) == 0 {
		return aArr, nil
	}
	if !aIsArr || len(aArr) == 0 {
		return bArr, nil
	}

	// Build index of B items by key
	bIndex := make(map[any]int)
	for i, item := range bArr {
		if obj, ok := item.(map[string]any); ok {
			if key, exists := obj[keyField]; exists {
				bIndex[key] = i
			}
		}
	}

	// Track which B items have been merged
	bMerged := make(map[int]bool)

	// Process A items, merging with B where keys match
	result := make([]any, 0, len(aArr)+len(bArr))
	for i, aItem := range aArr {
		aObj, aIsObj := aItem.(map[string]any)
		if !aIsObj {
			result = append(result, aItem)
			continue
		}

		aKey, aHasKey := aObj[keyField]
		if !aHasKey {
			result = append(result, aItem)
			continue
		}

		bIdx, bHasKey := bIndex[aKey]
		if !bHasKey {
			result = append(result, aItem)
			continue
		}

		// Handle matching key: either replace or deep merge
		if replaceOnMatch {
			// Replace: use A's item entirely, discard B's item
			result = append(result, aItem)
		} else {
			// Deep merge: A's fields override B's, but B's fields are preserved
			bItem := bArr[bIdx]
			itemPath := fmt.Sprintf("%s/%d", path, i)
			merged, err := m.deepMerge(aItem, bItem, itemPath)
			if err != nil {
				return nil, err
			}
			result = append(result, merged)
		}
		bMerged[bIdx] = true
	}

	// Add B items that weren't merged
	for i, bItem := range bArr {
		if !bMerged[i] {
			result = append(result, bItem)
		}
	}

	return result, nil
}

// mergeByDiscriminator merges two arrays of discriminated union objects.
// Items with matching discriminator values are deep merged (or replaced if replaceOnMatch is true);
// items only in A or B are included.
// This is useful for oneOf arrays where each object has a "type" field indicating its variant.
func (m *Merger) mergeByDiscriminator(a, b any, discriminatorField string, replaceOnMatch bool, path string) (any, error) {
	aArr, aIsArr := a.([]any)
	bArr, bIsArr := b.([]any)

	if !aIsArr && !bIsArr {
		return nil, fmt.Errorf("mergeByDiscriminator strategy requires arrays")
	}

	if !bIsArr || len(bArr) == 0 {
		return aArr, nil
	}
	if !aIsArr || len(aArr) == 0 {
		return bArr, nil
	}

	if discriminatorField == "" {
		discriminatorField = "type" // Default to "type" as discriminator
	}

	// Build index of B items by discriminator value
	// Note: For discriminated unions, we expect at most one item per discriminator value,
	// but we'll handle multiple by keeping the first one.
	bIndex := make(map[any]int)
	for i, item := range bArr {
		if obj, ok := item.(map[string]any); ok {
			if discValue, exists := obj[discriminatorField]; exists {
				if _, alreadyExists := bIndex[discValue]; !alreadyExists {
					bIndex[discValue] = i
				}
			}
		}
	}

	// Track which B items have been merged
	bMerged := make(map[int]bool)

	// Process A items, merging with B where discriminators match
	result := make([]any, 0, len(aArr)+len(bArr))
	for i, aItem := range aArr {
		aObj, aIsObj := aItem.(map[string]any)
		if !aIsObj {
			result = append(result, aItem)
			continue
		}

		aDiscValue, aHasDisc := aObj[discriminatorField]
		if !aHasDisc {
			result = append(result, aItem)
			continue
		}

		bIdx, bHasDisc := bIndex[aDiscValue]
		if !bHasDisc {
			// A has a new type that B doesn't have
			result = append(result, aItem)
			continue
		}

		// Handle matching discriminator: either replace or deep merge
		if replaceOnMatch {
			// Replace: use A's item entirely, discard B's item
			result = append(result, aItem)
		} else {
			// Deep merge: A's fields override B's, but B's fields are preserved
			bItem := bArr[bIdx]
			itemPath := fmt.Sprintf("%s/%d", path, i)
			merged, err := m.deepMerge(aItem, bItem, itemPath)
			if err != nil {
				return nil, err
			}
			result = append(result, merged)
		}
		bMerged[bIdx] = true
	}

	// Add B items that weren't merged (preserving types not in A)
	for i, bItem := range bArr {
		if !bMerged[i] {
			result = append(result, bItem)
		}
	}

	return result, nil
}

// overlay merges A into B, but only applies fields that A explicitly provides.
// Unlike deepMerge, overlay treats the entire A object as a partial update.
// Fields in A are applied to B; fields not in A are left unchanged from B.
// This is useful for PATCH-like semantics where A represents only the changes.
func (m *Merger) overlay(a, b any, path string) (any, error) {
	aMap, aIsMap := a.(map[string]any)
	bMap, bIsMap := b.(map[string]any)

	// If A is nil, use B
	if a == nil {
		return b, nil
	}

	// If B is nil, use A
	if b == nil {
		return a, nil
	}

	// If not both maps, A wins (like mergeRequest)
	if !aIsMap || !bIsMap {
		return a, nil
	}

	// Start with a copy of B
	result := make(map[string]any)
	for k, v := range bMap {
		result[k] = v
	}

	// Apply A's values as an overlay (only fields A has are considered)
	for k, aVal := range aMap {
		fieldPath := path + "/" + k
		bVal, bHasKey := bMap[k]

		// Check null handling for this field
		nullHandling := m.schema.NullHandlingFor(fieldPath)

		// If A's value is null and nullHandling is asAbsent, skip (keep B's value)
		if aVal == nil && nullHandling == "asAbsent" {
			continue
		}

		if !bHasKey {
			// A has a key B doesn't have
			result[k] = aVal
		} else {
			// Both have the key - recursively overlay if both are objects
			aValMap, aIsMap := aVal.(map[string]any)
			bValMap, bIsMap := bVal.(map[string]any)

			if aIsMap && bIsMap {
				// Recursively overlay nested objects
				merged, err := m.overlay(aValMap, bValMap, fieldPath)
				if err != nil {
					return nil, err
				}
				result[k] = merged
			} else {
				// A's value overwrites B's
				result[k] = aVal
			}
		}
	}

	return result, nil
}
