package kfsmerge

import "fmt"

// concatArrays concatenates two arrays. If unique is true, removes duplicate primitive values.
func (m *Merger) concatArrays(a, b any, unique bool) (any, error) {
	aArr, aIsArr := a.([]any)
	bArr, bIsArr := b.([]any)

	if !aIsArr && !bIsArr {
		return nil, fmt.Errorf("concat strategy requires arrays")
	}

	if !bIsArr {
		if unique {
			return m.deduplicateArray(aArr), nil
		}
		return aArr, nil
	}
	if !aIsArr {
		if unique {
			return m.deduplicateArray(bArr), nil
		}
		return bArr, nil
	}

	result := make([]any, 0, len(bArr)+len(aArr))
	result = append(result, bArr...)
	result = append(result, aArr...)

	if unique {
		return m.deduplicateArray(result), nil
	}
	return result, nil
}

// deduplicateArray removes duplicate primitive values from an array.
func (m *Merger) deduplicateArray(arr []any) []any {
	seen := make(map[any]bool)
	result := make([]any, 0, len(arr))
	for _, item := range arr {
		if isPrimitive(item) {
			if !seen[item] {
				seen[item] = true
				result = append(result, item)
			}
		} else {
			// Non-primitive values are always included
			result = append(result, item)
		}
	}
	return result
}

// numericOperation performs numeric operations (sum, max, min) on two values.
func (m *Merger) numericOperation(a, b any, operation string) (any, error) {
	aNum, aOk := toFloat64(a)
	bNum, bOk := toFloat64(b)

	if !aOk && !bOk {
		return nil, fmt.Errorf("numeric strategy requires numbers")
	}
	if !aOk {
		return b, nil
	}
	if !bOk {
		return a, nil
	}

	switch operation {
	case "sum":
		return aNum + bNum, nil
	case "max":
		if aNum > bNum {
			return a, nil
		}
		return b, nil
	case "min":
		if aNum < bNum {
			return a, nil
		}
		return b, nil
	default:
		return nil, fmt.Errorf("unknown numeric operation: %s", operation)
	}
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
