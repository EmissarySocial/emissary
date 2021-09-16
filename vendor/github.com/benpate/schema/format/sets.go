package format

import (
	"errors"
	"strings"
)

func In(arg string) StringFormat {

	options := strings.Split(arg, ",")

	return func(value string) error {

		for _, option := range options {
			if value == option {
				return nil
			}
		}

		return errors.New(value + " is not an allowed value")
	}
}

func NotIn(arg string) StringFormat {

	options := strings.Split(arg, ",")

	return func(value string) error {

		for _, option := range options {
			if value == option {
				return errors.New(value + " is not an allowed value")
			}
		}

		return nil
	}
}
