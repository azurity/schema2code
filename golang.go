package schema2code

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"github.com/azurity/schema2code/schemas"
	"io"
	"strings"
	"sync/atomic"
)

//go:embed helper_go
var helperCode []byte

type GolangConfig struct {
	CommonConfig
	Package string
}

type Context struct {
	regexCounter uint64
}

type Path struct {
	namedPath []string
	quotePath []string
}

func validationError(writer *CodeWriter, reason string) {
	writer.Write(fmt.Sprintf("return errors.New(%s)", reason))
	// TODO: add more log here
}

func formatName(name string) string {
	snake := strings.ReplaceAll(name, "-", "_")
	if snake == "" {
		return snake
	}
	return strings.ToUpper(snake[:1]) + snake[1:]
}

func generateNull(ctx *Context, path *Path, imports map[string]interface{}, desc *schemas.Type, optional bool, writer *CodeWriter, globalCode *CodeWriter, validationCode *CodeWriter) error {
	writer.Write("*null")
	return nil
}

func generateBoolean(ctx *Context, path *Path, imports map[string]interface{}, desc *schemas.Type, optional bool, writer *CodeWriter, globalCode *CodeWriter, validationCode *CodeWriter) error {
	if optional {
		writer.Write("*bool")
	} else {
		writer.Write("bool")
	}
	return nil
}

func generateInteger(ctx *Context, path *Path, imports map[string]interface{}, desc *schemas.Type, optional bool, writer *CodeWriter, globalCode *CodeWriter, validationCode *CodeWriter) error {
	if optional {
		writer.writer("*int")
	} else {
		writer.Write("int")
	}
	mini := float64(0)
	maxi := float64(0)
	hasMini := false
	hasMaxi := false
	exMini := false
	exMaxi := false
	if desc.Minimum != nil {
		hasMini = true
		mini = *desc.Minimum
		if desc.ExclusiveMinimum != nil {
			exMini = *desc.ExclusiveMinimum
		}
	}
	if desc.Maximum != nil {
		hasMaxi = true
		maxi = *desc.Maximum
		if desc.ExclusiveMaximum != nil {
			exMaxi = *desc.ExclusiveMaximum
		}
	}
	multiple := 1
	useMultiple := false
	if desc.MultipleOf != nil {
		useMultiple = true
		multiple = *desc.MultipleOf
	}
	if hasMini || hasMaxi || useMultiple {
		prefix := "&"
		if optional {
			prefix = ""
		}
		validationCode.CommonLine()
		validationCode.Write("if !")
		validationCode.Write(fmt.Sprintf("IntegerValidation(%g, %g, %t, %t, %t, %t, %d, %t, %s%s)", mini, maxi, hasMini, hasMaxi, exMini, exMaxi, multiple, useMultiple, prefix, strings.Join(path.namedPath, ".")))
		validationCode.Write(" {")
		validationCode.Indent()
		validationError(validationCode, "integer check failed")
		validationCode.Dedent()
		validationCode.Write("}")
	}
	return nil
}

func generateNumber(ctx *Context, path *Path, imports map[string]interface{}, desc *schemas.Type, optional bool, writer *CodeWriter, globalCode *CodeWriter, validationCode *CodeWriter) error {
	if optional {
		writer.writer("*float64")
	} else {
		writer.Write("float64")
	}
	mini := float64(0)
	maxi := float64(0)
	hasMini := false
	hasMaxi := false
	exMini := false
	exMaxi := false
	if desc.Minimum != nil {
		hasMini = true
		mini = *desc.Minimum
		if desc.ExclusiveMinimum != nil {
			exMini = *desc.ExclusiveMinimum
		}
	}
	if desc.Maximum != nil {
		hasMaxi = true
		maxi = *desc.Maximum
		if desc.ExclusiveMaximum != nil {
			exMaxi = *desc.ExclusiveMaximum
		}
	}
	multiple := 1
	useMultiple := false
	if desc.MultipleOf != nil {
		useMultiple = true
		multiple = *desc.MultipleOf
	}
	if hasMini || hasMaxi || useMultiple {
		prefix := "&"
		if optional {
			prefix = ""
		}
		validationCode.CommonLine()
		validationCode.Write("if !")
		validationCode.Write(fmt.Sprintf("NumberValidation(%g, %g, %t, %t, %t, %t, %d, %t, %s%s)", mini, maxi, hasMini, hasMaxi, exMini, exMaxi, multiple, useMultiple, prefix, strings.Join(path.namedPath, ".")))
		validationCode.Write(" {")
		validationCode.Indent()
		validationError(validationCode, "number check failed")
		validationCode.Dedent()
		validationCode.Write("}")
	}
	return nil
}

func generateString(ctx *Context, path *Path, imports map[string]interface{}, desc *schemas.Type, optional bool, writer *CodeWriter, globalCode *CodeWriter, validationCode *CodeWriter) error {
	if optional {
		writer.Write("*")
	}
	if desc.Format != nil {
		// TODO:
	}
	minLen := 0
	maxLen := 0
	useMinLength := false
	useMaxLength := false
	if desc.MinLength != nil {
		useMinLength = true
		minLen = *desc.MinLength
	}
	if desc.MaxLength != nil {
		useMaxLength = true
		maxLen = *desc.MaxLength
	}
	prefix := "&"
	if optional {
		prefix = ""
	}
	stringName := "raw"
	for _, it := range path.quotePath {
		stringName += "[" + it + "]"
	}
	if useMinLength || useMaxLength {
		validationCode.CommonLine()
		validationCode.Write("if !")
		validationCode.Write(fmt.Sprintf("StringValidation（%d， %d, %t, %t, %s%s)", minLen, maxLen, useMinLength, useMaxLength, prefix, stringName))
		validationCode.Write(" {")
		validationCode.Indent()
		validationError(validationCode, "string check length failed")
		validationCode.Dedent()
		validationCode.Write("}")
	}
	if desc.Pattern != nil {
		imports["regexp"] = struct{}{}
		index := atomic.AddUint64(&ctx.regexCounter, 1)
		globalCode.CommonLine()
		globalCode.Write(fmt.Sprintf("var stringRegex%d = regexp.MustCompile(`%s`)", index, *desc.Pattern))
		validationCode.CommonLine()
		validationCode.Write("if !")
		validationCode.Write(fmt.Sprintf("stringRegex%d.MatchString(%s%s)", index, prefix, stringName))
		validationCode.Write(" {")
		validationCode.Indent()
		validationError(validationCode, "string check pattern failed")
		validationCode.Dedent()
		validationCode.Write("}")
	}
	return nil
}

func generateArray(ctx *Context, path *Path, imports map[string]interface{}, desc *schemas.Type, optional bool, writer *CodeWriter, globalCode *CodeWriter, validationCode *CodeWriter) error {
	if desc.AdditionalItems != nil {
		return errors.New("only support single type array")
	}
	if desc.Items == nil {
		return errors.New("array must have item type")
	}
	writer.Write("[]")
	arrayName := strings.Join(path.namedPath, ".")
	if !optional {
		validationCode.CommonLine()
		validationCode.Write(fmt.Sprintf("if %s == nil {", arrayName))
		validationCode.Indent()
		validationError(validationCode, "array must have value")
		validationCode.Dedent()
		validationCode.Write("}")
	}
	validationCode.CommonLine()
	validationCode.Write(fmt.Sprintf("if %s != nil {", arrayName))
	validationCode.Indent()

	if desc.MinItems != nil || desc.MaxItems != nil {
		mini := 0
		maxi := 0
		if desc.MaxItems != nil {
			mini = *desc.MinItems
		}
		if desc.MaxItems != nil {
			maxi = *desc.MaxItems
		}
		validationCode.Write("if !")
		validationCode.Write(fmt.Sprintf("ArrayValidation(%d, %d, %t, %t, %t, %s)", mini, maxi, desc.MinItems != nil, desc.MaxItems != nil, desc.UniqueItems, arrayName))
		validationCode.Write(" {")
		validationCode.Indent()
		validationError(validationCode, "array check failed")
		validationCode.Dedent()
		validationCode.Write("}")
	}

	validationCode.Write(fmt.Sprintf("for index, item := range %s {", arrayName))
	validationCode.Indent()
	err := generateType(ctx, &Path{
		namedPath: append(append([]string{}, path.namedPath[:len(path.namedPath)-1]...), path.namedPath[len(path.namedPath)-1]+"[index]"),
		quotePath: append(append([]string{}, path.quotePath...), "index"),
	}, imports, desc.Items, false, writer, globalCode, validationCode)
	if err != nil {
		return err
	}
	validationCode.Dedent()
	validationCode.Write("}")
	validationCode.Dedent()
	validationCode.Write("}")
	return nil
}

func generateObject(ctx *Context, path *Path, imports map[string]interface{}, desc *schemas.Type, optional bool, writer *CodeWriter, globalCode *CodeWriter, validationCode *CodeWriter) error {
	if optional {
		writer.Write("*struct{")
	} else {
		writer.Write("struct{")
	}
	writer.Indent()

	required := desc.Required
	if required == nil {
		required = []string{}
	}

	validationCode.CommonLine()
	if optional {
		validationCode.Write(fmt.Sprintf("if %s != nil {", strings.Join(path.namedPath, ".")))
		validationCode.Indent()
	}

	for name, value := range desc.Properties {
		propOptional := false
		for _, item := range required {
			if name == item {
				propOptional = true
				break
			}
		}
		writer.CommonLine()
		writer.Write(fmt.Sprintf("%s ", name))
		err := generateType(ctx, &Path{
			namedPath: append(append([]string{}, path.namedPath...), name),
			quotePath: append(append([]string{}, path.quotePath...), "\""+name+"\""),
		}, imports, value, propOptional, writer, globalCode, validationCode)
		if err != nil {
			return err
		}
	}

	if optional {
		validationCode.Dedent()
		validationCode.Write("}")
	}

	writer.Dedent()
	writer.Write("}")
	return nil
}

func generateType(ctx *Context, path *Path, imports map[string]interface{}, desc *schemas.Type, optional bool, writer *CodeWriter, globalCode *CodeWriter, validationCode *CodeWriter) error {
	if desc == nil {
		return errors.New("must define type impl")
	}
	if len(desc.Type) != 1 {
		// TODO: try union later by use interface{}
		return errors.New("multiple type is not supported")
	}
	if desc.Ref != nil {
		parts := strings.Split(*desc.Ref, "/")
		if parts[0] != "#" {
			return errors.New("only local $ref is support")
		}
		parts = parts[1:]
		realName := []string{}
		for i, item := range parts {
			if i%2 == 0 {
				realName = append(realName, formatName(item))
			} else {
				if item != "$refs" && item != "definitions" {
					return errors.New("wrong $ref format")
				}
			}
		}
		if optional {
			writer.Write("*")
		}
		writer.Write(strings.Join(realName, ""))
		return nil
	}
	// TODO: impl enum here
	switch desc.Type[0] {
	case schemas.TypeNameNull:
		return generateNull(ctx, path, imports, desc, optional, writer, globalCode, validationCode)
	case schemas.TypeNameBoolean:
		return generateBoolean(ctx, path, imports, desc, optional, writer, globalCode, validationCode)
	case schemas.TypeNameInteger:
		return generateInteger(ctx, path, imports, desc, optional, writer, globalCode, validationCode)
	case schemas.TypeNameNumber:
		return generateNumber(ctx, path, imports, desc, optional, writer, globalCode, validationCode)
	case schemas.TypeNameString:
		return generateString(ctx, path, imports, desc, optional, writer, globalCode, validationCode)
	case schemas.TypeNameArray:
		return generateArray(ctx, path, imports, desc, optional, writer, globalCode, validationCode)
	case schemas.TypeNameObject:
		return generateObject(ctx, path, imports, desc, optional, writer, globalCode, validationCode)
	default:
		return errors.New(fmt.Sprintf("unknown type %s", desc.Type[0]))
	}
}

func generateGolangCode(types map[string]*TypeDesc, config *GolangConfig, writer io.Writer) error {
	for key, value := range types {
		rendered := []string{}
		for _, it := range value.Path {
			rendered = append(rendered, formatName(it))
		}
		types[key].RenderedName = strings.Join(rendered, "")
	}

	fileBuffer := &bytes.Buffer{}
	fileWriter := &CodeWriter{
		writer: fileBuffer,
		tab:    "\t",
	}

	imports := map[string]interface{}{
		"encoding/json": struct{}{},
		"errors":        struct{}{},
		"mah":           struct{}{},
		"regexp":        struct{}{},
	}

	ctx := Context{}

	for _, value := range types {
		typeBuffer := &bytes.Buffer{}
		typeWriter := &CodeWriter{
			writer: typeBuffer,
			tab:    "\t",
		}
		validationBuffer := &bytes.Buffer{}
		validationWriter := &CodeWriter{
			writer: validationBuffer,
			tab:    "\t",
		}
		typeWriter.Write(fmt.Sprintf("type %s ", value.RenderedName))
		validationWriter.Write(fmt.Sprintf("func (self *%s) UnmarshalJSON(buffer []byte) error {", value.RenderedName))
		validationWriter.Indent()
		validationWriter.Write("raw := map[string]interface{}{}")
		validationWriter.CommonLine()
		validationWriter.Write("err := json.Unmarshal(buffer, &raw)\n\tif err != nil {\n\t\treturn err\n\t}")
		validationWriter.CommonLine()
		validationWriter.Write(fmt.Sprintf("type internal %s", value.RenderedName))
		validationWriter.CommonLine()
		validationWriter.Write("main := internal{}\n\terr = json.Unmarshal(buffer, &main)\n\tif err != nil {\n\t\treturn err\n\t}")
		validationWriter.CommonLine()

		err := generateType(&ctx, &Path{
			namedPath: []string{"main"},
			quotePath: []string{},
		}, imports, value.Type, false, typeWriter, fileWriter, validationWriter)
		if err != nil {
			return err
		}

		validationWriter.CommonLine()
		validationWriter.Write(fmt.Sprintf("*self = %s(main)", value.RenderedName))
		validationWriter.CommonLine()
		validationWriter.Write("return nil")
		validationWriter.Dedent()
		validationWriter.Write("}")

		fileWriter.CommonLine()
		fileWriter.writer.Write(typeBuffer.Bytes())
		fileWriter.CommonLine()
		fileWriter.writer.Write(validationBuffer.Bytes())
		fileWriter.CommonLine()
	}

	packageParts := strings.Split(config.Package, "/")
	packName := packageParts[len(packageParts)-1]
	writer.Write([]byte(fmt.Sprintf("package %s\n\n", packName)))
	writer.Write([]byte("import (\n"))
	for pack, _ := range imports {
		writer.Write([]byte(fmt.Sprintf("\t\"%s\"\n", pack)))
	}
	writer.Write([]byte(")\n\n"))
	writer.Write(helperCode)
	writer.Write(fileBuffer.Bytes())

	return nil
}
