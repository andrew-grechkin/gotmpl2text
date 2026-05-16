package main

import (
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/google/uuid"
)

// additionalFuncMap returns template functions for additional template functions not yet provided by Sprig
func additionalFuncMap() template.FuncMap {
	return template.FuncMap{
		"toJson": func(v any) string {
			data, err := json.Marshal(v)
			if err != nil {
				return ""
			}
			return string(data)
		},

		"uuidv7": func() string {
			return uuid.Must(uuid.NewV7()).String()
		},

		"uuidv7ToEpochNs": func(uuidStr string) (int64, error) {
			id, err := uuid.Parse(uuidStr)
			if err != nil {
				return 0, fmt.Errorf("invalid UUID: %w", err)
			}
			sec, nsec := id.Time().UnixTime()
			return sec*1000000000 + nsec, nil
		},

		"uuidv7ToEpoch": func(uuidStr string) (int64, error) {
			id, err := uuid.Parse(uuidStr)
			if err != nil {
				return 0, fmt.Errorf("invalid UUID: %w", err)
			}
			sec, _ := id.Time().UnixTime()
			return sec, nil
		},
	}
}
