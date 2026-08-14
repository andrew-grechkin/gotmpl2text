// Package uuid provides UUID v7 (time-ordered) generation and conversion helpers - fills the gap left by Go's stdlib
// and sprig, which only ship v4
package uuid

import (
	"fmt"
	"text/template"

	guuid "github.com/google/uuid"
)

const nsPerSecond = 1_000_000_000

func FuncMap() template.FuncMap {
	return template.FuncMap{
		"uuidv7": func() (string, error) {
			id, err := guuid.NewV7()
			if err != nil {
				return "", fmt.Errorf("uuidv7: %w", err)
			}
			return id.String(), nil
		},

		"uuidv7ToEpochNs": func(uuidStr string) (int64, error) {
			id, err := guuid.Parse(uuidStr)
			if err != nil {
				return 0, fmt.Errorf("invalid UUID: %w", err)
			}
			sec, nsec := id.Time().UnixTime()
			return sec*nsPerSecond + nsec, nil
		},

		"uuidv7ToEpoch": func(uuidStr string) (int64, error) {
			id, err := guuid.Parse(uuidStr)
			if err != nil {
				return 0, fmt.Errorf("invalid UUID: %w", err)
			}
			sec, _ := id.Time().UnixTime()
			return sec, nil
		},
	}
}
