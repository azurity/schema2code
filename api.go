package schema2code

import (
	"errors"
	"fmt"
	"github.com/azurity/schema2code/schemas"
	"io"
	"strings"
)

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

func walkDefs(baseKey []string, defs schemas.Definitions, action func(key []string, item *schemas.Type) error) error {
	if defs == nil {
		return nil
	}
	for key, item := range defs {
		newKey := append(baseKey, key)
		if err := action(append([]string{}, newKey...), item); err != nil {
			return err
		}
		if err := walkDefs(newKey, item.Definitions, action); err != nil {
			return err
		}
	}
	return nil
}

//func walkInlineType(index *uint64, item *schemas.Type, action func(index uint64, item *schemas.Type)) {
//	// TODO:
//}

func Generate(reader io.Reader, writer io.Writer, config interface{}) error {
	schema, err := schemas.FromJSONReader(reader)
	if err != nil {
		return err
	}

	casedConfig := config.(IConfig).Common()
	types := map[string]*TypeDesc{}
	if schema.ObjectAsType != nil {
		if casedConfig.RootType == "" {
			return errors.New("need a root-type name")
		}
		rootType := schemas.Type(*schema.ObjectAsType)
		types[casedConfig.RootType] = &TypeDesc{
			Path: []string{},
			Type: &rootType,
		}
	}
	err = walkDefs([]string{}, schema.Definitions, func(key []string, item *schemas.Type) error {
		unifiedName := strings.Join(key, "/")
		if _, ok := types[unifiedName]; ok {
			return errors.New(fmt.Sprintf("duplicate name %s", unifiedName))
		}
		types[unifiedName] = &TypeDesc{
			Path: key,
			Type: item,
		}
		return nil
	})
	if err != nil {
		return err
	}

	switch config.(type) {
	case *GolangConfig:
		return generateGolangCode(types, config.(*GolangConfig), writer)
	default:
		return errors.New("unknown config type")
	}
}
