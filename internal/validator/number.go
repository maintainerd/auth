package validator

import (
	"errors"
	"fmt"
)

func Number() FieldRule {
	return FieldRule{
		rule: func(value any) error {
			switch value.(type) {
			case int, int64, float32, float64:
				return nil
			default:
				return errors.New("must be a number")
			}
		},
	}
}

func Min(min float64) FieldRule {
	return FieldRule{
		rule: func(value any) error {
			num := toFloat64(value)
			if num < min {
				return fmt.Errorf("must be >= %.2f", min)
			}
			return nil
		},
	}
}

func Max(max float64) FieldRule {
	return FieldRule{
		rule: func(value any) error {
			num := toFloat64(value)
			if num > max {
				return fmt.Errorf("must be <= %.2f", max)
			}
			return nil
		},
	}
}

func toFloat64(value any) float64 {
	switch v := value.(type) {
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case float32:
		return float64(v)
	case float64:
		return v
	default:
		return 0
	}
}
