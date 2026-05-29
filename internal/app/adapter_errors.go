package app

import (
	"errors"

	"gorm.io/gorm"
)

func firstOrNil(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	return err
}
