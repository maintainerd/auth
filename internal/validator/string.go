package validator

import (
	"errors"
	"net/mail"
	"regexp"
)

func String() FieldRule {
	return FieldRule{
		rule: func(value any) error {
			if _, ok := value.(string); !ok {
				return errors.New("must be a string")
			}
			return nil
		},
	}
}

func Email() FieldRule {
	return FieldRule{
		rule: func(value any) error {
			s, ok := value.(string)
			if !ok {
				return errors.New("must be a string")
			}
			if _, err := mail.ParseAddress(s); err != nil {
				return errors.New("invalid email format")
			}
			return nil
		},
	}
}

func Match(pattern *regexp.Regexp) FieldRule {
	return FieldRule{
		rule: func(value any) error {
			s, ok := value.(string)
			if !ok {
				return errors.New("must be a string")
			}
			if !pattern.MatchString(s) {
				return errors.New("invalid format")
			}
			return nil
		},
	}
}
