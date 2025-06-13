package validator

import (
	"errors"
	"fmt"
)

func Array() FieldRule {
	return FieldRule{
		rule: func(value any) error {
			switch value.(type) {
			case []any, []string, []int, []float64, []int64:
				return nil
			default:
				return errors.New("must be an array")
			}
		},
	}
}

func MinItems(min int) FieldRule {
	return FieldRule{
		rule: func(value any) error {
			length := getArrayLength(value)
			if length == -1 {
				return errors.New("must be an array")
			}
			if length < min {
				return fmt.Errorf("must have at least %d items", min)
			}
			return nil
		},
	}
}

func MaxItems(max int) FieldRule {
	return FieldRule{
		rule: func(value any) error {
			length := getArrayLength(value)
			if length == -1 {
				return errors.New("must be an array")
			}
			if length > max {
				return fmt.Errorf("must have at most %d items", max)
			}
			return nil
		},
	}
}

func getArrayLength(value any) int {
	switch v := value.(type) {
	case []any:
		return len(v)
	case []string:
		return len(v)
	case []int:
		return len(v)
	case []float64:
		return len(v)
	case []int64:
		return len(v)
	default:
		return -1
	}
}
