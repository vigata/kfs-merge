package kfsmerge

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
		if isPrimitive(item) {
			if !seen[item] {
				seen[item] = true
				result = append(result, item)
			}
		} else {
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

// mergeByDiscriminator merges two arrays of objects by a discriminator field.
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
		discriminatorField = "type"
	}

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

	bMerged := make(map[int]bool)
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
			result = append(result, aItem)
			continue
		}

		if replaceOnMatch {
			result = append(result, aItem)
		} else {
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

	for i, bItem := range bArr {
		if !bMerged[i] {
			result = append(result, bItem)
		}
	}

	return result, nil
}

// overlay merges A into B, but only applies fields that A explicitly provides.
func (m *Merger) overlay(a, b any, path string) (any, error) {
	aMap, aIsMap := a.(map[string]any)
	bMap, bIsMap := b.(map[string]any)

	if a == nil {
		return b, nil
	}
	if b == nil {
		return a, nil
	}

	if !aIsMap || !bIsMap {
		return a, nil
	}

	result := make(map[string]any)
	for k, v := range bMap {
		result[k] = v
	}

	for k, aVal := range aMap {
		fieldPath := path + "/" + k
		bVal, bHasKey := bMap[k]

		nullHandling := m.schema.NullHandlingFor(fieldPath)
		if aVal == nil && nullHandling == "asAbsent" {
			continue
		}

		if !bHasKey {
			result[k] = aVal
		} else {
			aValMap, aIsMap := aVal.(map[string]any)
			bValMap, bIsMap := bVal.(map[string]any)

			if aIsMap && bIsMap {
				merged, err := m.overlay(aValMap, bValMap, fieldPath)
				if err != nil {
					return nil, err
				}
				result[k] = merged
			} else {
				result[k] = aVal
			}
		}
	}

	return result, nil
}
