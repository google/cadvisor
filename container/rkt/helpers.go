package rkt

import (
	"fmt"
	"strings"
)

type parsedName struct {
	Pod       string
	Container string
}

func verifyName(name string) (bool, error) {
	_, err := parseName(name)
	if err != nil {
		return false, err
	}
	return true, nil
}

func parseName(name string) (*parsedName, error) {
	splits := strings.Split(name, "/")
	if len(splits) == 3 || len(splits) == 5 {
		parsed := &parsedName{}

		if splits[1] == "machine.slice" {
			replacer := strings.NewReplacer("machine-rkt\\x2d", "", ".scope", "", "\\x2d", "-")
			parsed.Pod = replacer.Replace(splits[2])
			if len(splits) == 3 {
				return parsed, nil
			}
			if splits[3] == "system.slice" {
				parsed.Container = strings.Replace(splits[4], ".service", "", -1)
				return parsed, nil
			}
		}
	}

	return nil, fmt.Errorf("%s not handled by rkt handler", name)
}
