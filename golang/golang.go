package golang

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"github.com/azurity/schema2code/common"
	"github.com/azurity/schema2code/schemas"
	"io"
	"sort"
	"strings"
	"sync/atomic"
)

//go:embed helper_go
var helperCode []byte

type Config struct {
	common.CommonConfig
	Package string
}

type Context struct {
	regexCounter uint64
}

type Path struct {
	namedPath []string
	quotePath []string
}

func validationError(writer *common.CodeWriter, reason string) {
	writer.Write(fmt.Sprintf("return errors.New(\"%s\")", reason))
	// TODO: add more log here
}

func formatName(name string) string {
	snake := strings.ReplaceAll(name, "-", "_")
	if snake == "" {
		return snake
	}
	return strings.ToUpper(snake[:1]) + snake[1:]
}

func generateNull(ctx *Context, path *Path, imports map[string]interface{}, desc *schemas.Type, optional bool, writer *common.CodeWriter, globalCode *common.CodeWriter, validationCode *common.CodeWriter) (bool, error) {
	writer.Write("*null")
	return true, nil
}

func generateBoolean(ctx *Context, path *Path, imports map[string]interface{}, desc *schemas.Type, optional bool, writer *common.CodeWriter, globalCode *common.CodeWriter, validationCode *common.CodeWriter) (bool, error) {
	if optional {
		writer.Write("*bool")
	} else {
		writer.Write("bool")
	}
	return true, nil
}

func generateInteger(ctx *Context, path *Path, imports map[string]interface{}, desc *schemas.Type, optional bool, writer *common.CodeWriter, globalCode *common.CodeWriter, validationCode *common.CodeWriter) (bool, error) {
	if optional {
		writer.Write("*int")
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
	return !(hasMini || hasMaxi || useMultiple), nil
}

func generateNumber(ctx *Context, path *Path, imports map[string]interface{}, desc *schemas.Type, optional bool, writer *common.CodeWriter, globalCode *common.CodeWriter, validationCode *common.CodeWriter) (bool, error) {
	if optional {
		writer.Write("*float64")
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
	return !(hasMini || hasMaxi || useMultiple), nil
}

func generateString(ctx *Context, path *Path, imports map[string]interface{}, desc *schemas.Type, optional bool, writer *common.CodeWriter, globalCode *common.CodeWriter, validationCode *common.CodeWriter) (bool, error) {
	if optional {
		writer.Write("*")
	}
	if desc.Format != nil {
		// TODO:
	}
	writer.Write("string")
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
		validationCode.Write(fmt.Sprintf("StringValidation(%d, %d, %t, %t, %s%s)", minLen, maxLen, useMinLength, useMaxLength, prefix, stringName))
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
	return !(useMinLength || useMaxLength || desc.Pattern != nil), nil
}

func generateArray(ctx *Context, path *Path, imports map[string]interface{}, desc *schemas.Type, optional bool, writer *common.CodeWriter, globalCode *common.CodeWriter, validationCode *common.CodeWriter) (bool, error) {
	if desc.AdditionalItems != nil {
		return false, errors.New("only support single type array")
	}
	if desc.Items == nil {
		return false, errors.New("array must have item type")
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
	_, ignore, err := generateType(ctx, &Path{
		namedPath: []string{"item"},
		quotePath: append(append([]string{}, path.quotePath...), "index"),
	}, imports, desc.Items, false, writer, globalCode, validationCode)
	if err != nil {
		return false, err
	}
	validationCode.Dedent()
	validationCode.Write("}")
	validationCode.Dedent()
	validationCode.Write("}")
	return ignore && !(desc.MinItems != nil || desc.MaxItems != nil), nil
}

type sortableKV struct {
	key   string
	value interface{}
}

type sortKV []sortableKV

func (a sortKV) Len() int           { return len(a) }
func (a sortKV) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a sortKV) Less(i, j int) bool { return a[i].key < a[j].key }

func generateObject(ctx *Context, path *Path, imports map[string]interface{}, desc *schemas.Type, optional bool, writer *common.CodeWriter, globalCode *common.CodeWriter, validationCode *common.CodeWriter) (bool, error) {
	if optional {
		writer.Write("*struct{")
	} else {
		writer.Write("struct{")
	}
	writer.Indent()

	globalIgnore := true

	required := desc.Required
	if required == nil {
		required = []string{}
	}

	validationCode.CommonLine()
	if optional {
		validationCode.Write(fmt.Sprintf("if %s != nil {", strings.Join(path.namedPath, ".")))
		validationCode.Indent()
	}

	sorted := sortKV{}
	for name, value := range desc.Properties {
		sorted = append(sorted, sortableKV{name, value})
	}
	sort.Sort(sorted)

	for _, iter := range sorted {
		name := iter.key
		value := iter.value.(*schemas.Type)
		propOptional := true
		for _, item := range required {
			if name == item {
				propOptional = false
				break
			}
		}
		writer.CommonLine()
		writer.Write(fmt.Sprintf("%s ", formatName(name)))
		_, ignore, err := generateType(ctx, &Path{
			namedPath: append(append([]string{}, path.namedPath...), formatName(name)),
			quotePath: append(append([]string{}, path.quotePath...), "\""+name+"\""),
		}, imports, value, propOptional, writer, globalCode, validationCode)
		if err != nil {
			return false, err
		}

		globalIgnore = globalIgnore && ignore

		writer.Write(fmt.Sprintf(" `json:\"%s\"`", name))
	}

	if optional {
		validationCode.Dedent()
		validationCode.Write("}")
	}

	writer.Dedent()
	writer.Write("}")
	return globalIgnore, nil
}

// ignore value & error
func generateType(ctx *Context, path *Path, imports map[string]interface{}, desc *schemas.Type, optional bool, writer *common.CodeWriter, globalCode *common.CodeWriter, validationCode *common.CodeWriter) (string, bool, error) {
	if desc == nil {
		return "", false, errors.New("must define type impl")
	}
	if desc.Ref != nil {
		parts := strings.Split(*desc.Ref, "/")
		if parts[0] != "#" {
			return "", false, errors.New("only local $ref is support")
		}
		parts = parts[1:]
		realName := []string{}
		for i, item := range parts {
			if i%2 != 0 {
				realName = append(realName, formatName(item))
			} else {
				if item != "$defs" && item != "definitions" {
					return "", false, errors.New("wrong $ref format")
				}
			}
		}
		if optional {
			writer.Write("*")
		}
		writer.Write(strings.Join(realName, ""))
		return "", true, nil
	}
	if len(desc.Type) != 1 {
		// TODO: try union later by use interface{}
		return "", false, errors.New("multiple type is not supported")
	}
	// TODO: impl enum here
	switch desc.Type[0] {
	case schemas.TypeNameNull:
		ign, err := generateNull(ctx, path, imports, desc, optional, writer, globalCode, validationCode)
		return "", ign, err
	case schemas.TypeNameBoolean:
		ign, err := generateBoolean(ctx, path, imports, desc, optional, writer, globalCode, validationCode)
		return "", ign, err
	case schemas.TypeNameInteger:
		ign, err := generateInteger(ctx, path, imports, desc, optional, writer, globalCode, validationCode)
		return "0", ign, err
	case schemas.TypeNameNumber:
		ign, err := generateNumber(ctx, path, imports, desc, optional, writer, globalCode, validationCode)
		return "float64(0)", ign, err
	case schemas.TypeNameString:
		ign, err := generateString(ctx, path, imports, desc, optional, writer, globalCode, validationCode)
		return "\"\"", ign, err
	case schemas.TypeNameArray:
		ign, err := generateArray(ctx, path, imports, desc, optional, writer, globalCode, validationCode)
		return "[]interface{}{}", ign, err
	case schemas.TypeNameObject:
		ign, err := generateObject(ctx, path, imports, desc, optional, writer, globalCode, validationCode)
		return "map[string]interface{}{}", ign, err
	default:
		return "", false, errors.New(fmt.Sprintf("unknown type %s", desc.Type[0]))
	}
}

func GenerateCode(types map[string]*common.TypeDesc, config *Config, writer io.Writer) error {
	for key, value := range types {
		rendered := []string{}
		for _, it := range value.Path {
			rendered = append(rendered, formatName(it))
		}
		types[key].RenderedName = strings.Join(rendered, "")
	}

	fileBuffer := &bytes.Buffer{}
	fileWriter := &common.CodeWriter{
		Writer: fileBuffer,
		Tab:    "\t",
	}

	imports := map[string]interface{}{
		"encoding/json": struct{}{},
		"errors":        struct{}{},
		"math":          struct{}{},
		"regexp":        struct{}{},
	}

	ctx := Context{}

	sortedType := sortKV{}
	for name, value := range types {
		sortedType = append(sortedType, sortableKV{name, value})
	}
	sort.Sort(sortedType)

	for _, iter := range sortedType {
		value := iter.value.(*common.TypeDesc)
		if value.Type.Enum != nil {
			if len(value.Type.Type) != 1 || value.Type.Type[0] != schemas.TypeNameString {
				return errors.New("only support string enum")
			}

			fileWriter.CommonLine()
			fileWriter.Write(fmt.Sprintf("type %s string", value.RenderedName))
			fileWriter.CommonLine()
			fileWriter.Write("const (")
			fileWriter.Indent()
			for _, item := range value.Type.Enum {
				if cased, ok := item.(string); !ok {
					return errors.New("only support string enum")
				} else {
					fileWriter.CommonLine()
					fileWriter.Write(fmt.Sprintf("%s%s %s = \"%s\"", value.RenderedName, formatName(cased), value.RenderedName, cased))
				}
			}
			fileWriter.Dedent()
			fileWriter.Write(")")
			fileWriter.CommonLine()
			fileWriter.Write(fmt.Sprintf("var enumValues%s = []string{", value.RenderedName))
			for i, item := range value.Type.Enum {
				if i != 0 {
					fileWriter.Write(", ")
				}
				fileWriter.Write(fmt.Sprintf("\"%s\"", item.(string)))
			}
			fileWriter.Write("}")
			fileWriter.CommonLine()
			fileWriter.Write(fmt.Sprintf("func (object *%s) UnmarshalJSON(buffer []byte) error {", value.RenderedName))
			fileWriter.Indent()

			fileWriter.Write("raw := \"\"")
			fileWriter.CommonLine()
			fileWriter.Write("err := json.Unmarshal(buffer, &raw)\n\tif err != nil {\n\t\treturn err\n\t}")
			fileWriter.CommonLine()
			fileWriter.Write(fmt.Sprintf("if !EnumValidation(raw, enumValues%s) {", value.RenderedName))
			fileWriter.Indent()
			validationError(fileWriter, "wrong enum value")
			fileWriter.Dedent()
			fileWriter.Write("}")
			fileWriter.CommonLine()

			fileWriter.Write(fmt.Sprintf("*object = %s(raw)", value.RenderedName))
			fileWriter.CommonLine()
			fileWriter.Write("return nil")
			fileWriter.Dedent()
			fileWriter.Write("}")
			fileWriter.CommonLine()
			continue
		}

		typeBuffer := &bytes.Buffer{}
		typeWriter := &common.CodeWriter{
			Writer: typeBuffer,
			Tab:    "\t",
		}
		validationBuffer := &bytes.Buffer{}
		validationWriter := &common.CodeWriter{
			Writer: validationBuffer,
			Tab:    "\t",
		}
		typeWriter.Write(fmt.Sprintf("type %s ", value.RenderedName))

		validationWriter.Indent()

		rawType, ignore, err := generateType(&ctx, &Path{
			namedPath: []string{"main"},
			quotePath: []string{},
		}, imports, value.Type, false, typeWriter, fileWriter, validationWriter)
		if err != nil {
			return err
		}

		fileWriter.CommonLine()
		fileWriter.Writer.Write(typeBuffer.Bytes())
		fileWriter.CommonLine()
		if !ignore {
			fileWriter.Write(fmt.Sprintf("func (object *%s) UnmarshalJSON(buffer []byte) error {", value.RenderedName))
			fileWriter.Indent()
			fileWriter.Write(fmt.Sprintf("raw := %s", rawType))
			fileWriter.CommonLine()
			fileWriter.Write("err := json.Unmarshal(buffer, &raw)\n\tif err != nil {\n\t\treturn err\n\t}")
			fileWriter.CommonLine()
			fileWriter.Write(fmt.Sprintf("type internal %s", value.RenderedName))
			fileWriter.CommonLine()
			fileWriter.Write("main := new(internal)\n\terr = json.Unmarshal(buffer, main)\n\tif err != nil {\n\t\treturn err\n\t}")

			fileWriter.Writer.Write(validationBuffer.Bytes())
			fileWriter.CommonLine()

			fileWriter.Write(fmt.Sprintf("*object = %s(*main)", value.RenderedName))
			fileWriter.CommonLine()
			fileWriter.Write("return nil")
			fileWriter.Dedent()
			fileWriter.Write("}")
		}
	}

	packageParts := strings.Split(config.Package, "/")
	packName := packageParts[len(packageParts)-1]
	writer.Write([]byte(fmt.Sprintf("package %s\n\n", packName)))
	writer.Write([]byte("import (\n"))

	sortedPack := []string{}
	for pack, _ := range imports {
		sortedPack = append(sortedPack, pack)
	}
	sort.Strings(sortedPack)

	for _, pack := range sortedPack {
		writer.Write([]byte(fmt.Sprintf("\t\"%s\"\n", pack)))
	}
	writer.Write([]byte(")\n\n"))
	writer.Write(helperCode)
	writer.Write(fileBuffer.Bytes())

	return nil
}
