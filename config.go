package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	BuiltinConfig = "config.builtin.yml"
)

type Config []*RepositoryConfig

type RepositoryConfig struct {
	Repository  string            `yaml:"repository"`
	Permissions *PermissionConfig `yaml:"permissions"`
}

type PermissionConfig map[string]string

func (c Config) GetRepositoryConfig(r string) *RepositoryConfig {
	for _, rc := range c {
		if rc.Repository == r {
			return rc
		}
	}

	return nil
}

func loadBuiltinConfig() (*Config, error) {
	f, err := os.Open(BuiltinConfig)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return loadConfig(f)
}

func loadConfig(r io.Reader) (*Config, error) {
	var c Config

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}

	if err := c.Validate(); err != nil {
		return nil, err
	}

	return &c, nil
}

func (c Config) Validate() error {
	for _, rc := range c {
		if len(strings.Split(rc.Repository, "/")) < 2 {
			return fmt.Errorf("invalid repository name: %s", rc.Repository)
		}

		if err := rc.Permissions.validate(); err != nil {
			return err
		}
	}

	return nil
}

func (c PermissionConfig) validate() error {
	for k, v := range c {
		if _, ok := allPermissionScopes[k]; !ok {
			return fmt.Errorf("invalid permission scope: %s", k)
		}

		switch v {
		case NoScope:
		case ReadScope:
		case WriteScope:
		default:
			return fmt.Errorf("invalid permission value: %s", v)
		}
	}

	return nil
}

const (
	NoScope    = ""
	ReadScope  = "read"
	WriteScope = "write"
)

var allPermissionScopes = map[string]struct{}{
	"actions":             {},
	"checks":              {},
	"contents":            {},
	"deployments":         {},
	"id-token":            {},
	"issues":              {},
	"discussions":         {},
	"packages":            {},
	"pages":               {},
	"pull-requests":       {},
	"repository-projects": {},
	"security-events":     {},
	"statuses":            {},
}
