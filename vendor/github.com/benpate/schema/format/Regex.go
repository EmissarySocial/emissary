package format

import (
	"errors"
	"regexp"
)

func MatchRegex(arg string) StringFormat {

	rx, err := regexp.Compile(arg)

	return func(value string) error {

		if err != nil {
			return err
		}

		if rx.Match([]byte(value)) {
			return nil
		}

		return errors.New("Does not match Regular Expression: " + arg)
	}
}
