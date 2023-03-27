package common

import "github.com/azurity/schema2code/schemas"

type CommonConfig struct {
	RootType string
}

type IConfig interface {
	Common() *CommonConfig
}

func (c *CommonConfig) Common() *CommonConfig {
	return c
}

type TypeDesc struct {
	Path         []string
	RenderedName string
	Type         *schemas.Type
}
