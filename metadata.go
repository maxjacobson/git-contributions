package main

import (
	"encoding/json"
	"os"
)

// Profile is user-maintained metadata about one contributor.
type Profile struct {
	Note    string   `json:"note,omitempty"`
	Active  bool     `json:"active,omitempty"`
	Team    string   `json:"team,omitempty"`
	Aliases []string `json:"aliases,omitempty"`
}

// Metadata maps contributor names (and aliases) to profiles, stored per repo
// so different repositories never affect each other.
type Metadata map[string]Profile

// LoadMetadata reads the metadata file. A missing file yields empty
// metadata; a malformed one yields an error the caller can surface.
func LoadMetadata(path string) (Metadata, error) {
	metadata := Metadata{}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return metadata, nil
	}
	if err != nil {
		return metadata, err
	}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return metadata, err
	}
	return metadata, nil
}

// Lookup finds the profile for a contributor by primary name or alias.
func (m Metadata) Lookup(name string) (Profile, bool) {
	if p, ok := m[name]; ok {
		return p, true
	}
	for _, p := range m {
		for _, alias := range p.Aliases {
			if alias == name {
				return p, true
			}
		}
	}
	return Profile{}, false
}
