package validator

import (
	"errors"
	"fmt"
)

/**
 * applies a single validation rule to each item in an array. Uses getArrayAsSlice() to normalize supported array types into []any.
 */
func Each(rule FieldRule) FieldRule {
	return FieldRule{
		rule: func(value any) error {
			slice := getArrayAsSlice(value)
			if slice == nil {
				return errors.New("must be an array")
			}
			for idx, item := range slice {
				if err := rule.Apply(item); err != nil {
					return fmt.Errorf("item[%d]: %w", idx, err)
				}
			}
			return nil
		},
	}
}

func getArrayAsSlice(value any) []any {
	switch v := value.(type) {
	case []any:
		return v
	case []string:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = item
		}
		return result
	case []int:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = item
		}
		return result
	case []float64:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = item
		}
		return result
	case []int64:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = item
		}
		return result
	default:
		return nil
	}
}
