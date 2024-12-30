package test

import (
	"fmt"
	"regexp"
	"testing"
)

type StringMatch struct {
	Exact       *string     `json:"exact"`
	Absent      *bool       `json:"absent"`
	Regex       *string     `json:"regex"`
	MatchAction MatchAction `json:"matchAction"`
}

func (sm StringMatch) Assert(t *testing.T, values ...string) bool {
	switch sm.MatchAction {
	case "", MatchActionFirst:
		var value string
		if len(values) > 0 {
			value = values[0]
		}
		return sm.match(value)
	case MatchActionAny:
		for _, value := range values {
			if sm.match(value) {
				return true
			}
		}
		return false
	case MatchActionAll:
		if len(values) == 0 {
			return false
		}
		for _, value := range values {
			if !sm.match(value) {
				return false
			}
		}
		return true
	}
	return false
}

func (sm *StringMatch) MatchType() string {
	switch {
	case sm.Exact != nil:
		return "exact"
	case sm.Absent != nil:
		return "absent"
	case sm.Regex != nil:
		return "regex"
	}
	return ""
}

func (sm *StringMatch) MatchValue() string {
	switch {
	case sm.Exact != nil:
		return *sm.Exact
	case sm.Absent != nil:
		return fmt.Sprintf("%t", *sm.Absent)
	case sm.Regex != nil:
		return *sm.Regex
	}

	return ""
}

func (sm *StringMatch) match(value string) bool {
	switch {
	case sm.Absent != nil:
		if *sm.Absent {
			return value == ""
		}
		return value != ""
	case sm.Exact != nil:
		return value == *sm.Exact
	case sm.Regex != nil:
		r := regexp.MustCompile(*sm.Regex)
		return r.MatchString(value)
	}
	return false
}
