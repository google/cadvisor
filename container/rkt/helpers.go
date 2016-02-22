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

//FIXME: need to figure out the \x2d situation
func parseName(name string) (*parsedName, error) {
	splits := strings.Split(name, "/")
	if len(splits) == 3 || len(splits) == 5 {
		parsed := &parsedName{}

		if splits[1] == "machine.slice" {
			machine_split := strings.Split(splits[2], ".scope")
			parsed.Pod = machine_split[0]
			if len(splits) == 3 {
				return parsed, nil
			}
			if splits[3] == "system.slice" {
				container_split := strings.Split(splits[4], ".scope")
				parsed.Container = container_split[0]
				return parsed, nil
			}
		}
	}

	return nil, fmt.Errorf("%s not handled by rkt handler", name)
}
