package typescript

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
)

//go:embed helper_ts
var helperCode []byte

type TypescriptConfig struct {
	common.CommonConfig
	//Package string
}

type Context struct {
	regexCounter uint64
}

type Path struct {
	namedPath []string
	//quotePath []string
}

func validationError(writer *common.CodeWriter, reason string) {
	writer.Write(fmt.Sprintf("throw new Error(\"%s\");", reason))
	// TODO: add more log here
}

func formatName(name string) string {
	snake := strings.ReplaceAll(name, "-", "_")
	if snake == "" {
		return snake
	}
	return strings.ToUpper(snake[:1]) + snake[1:]
}

func generateNull(ctx *Context, path *Path, desc *schemas.Type, writer *common.CodeWriter, globalCode *common.CodeWriter, validationCode *common.CodeWriter) (bool, error) {
	writer.Write("null")
	return true, nil
}

func generateBoolean(ctx *Context, path *Path, desc *schemas.Type, writer *common.CodeWriter, globalCode *common.CodeWriter, validationCode *common.CodeWriter) (bool, error) {
	writer.Write("boolean")
	return true, nil
}

func generateInteger(ctx *Context, path *Path, desc *schemas.Type, writer *common.CodeWriter, globalCode *common.CodeWriter, validationCode *common.CodeWriter) (bool, error) {
	writer.Write("number")
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
		validationCode.CommonLine()
		validationCode.Write("if (!")
		validationCode.Write(fmt.Sprintf("integerValidation(%g, %g, %t, %t, %t, %t, %d, %t, %s)", mini, maxi, hasMini, hasMaxi, exMini, exMaxi, multiple, useMultiple, strings.Join(path.namedPath, "")))
		validationCode.Write(") {")
		validationCode.Indent()
		validationError(validationCode, "integer check failed")
		validationCode.Dedent()
		validationCode.Write("}")
	}
	return !(hasMini || hasMaxi || useMultiple), nil
}

func generateNumber(ctx *Context, path *Path, desc *schemas.Type, writer *common.CodeWriter, globalCode *common.CodeWriter, validationCode *common.CodeWriter) (bool, error) {
	writer.Write("number")
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
		validationCode.CommonLine()
		validationCode.Write("if (!")
		validationCode.Write(fmt.Sprintf("numberValidation(%g, %g, %t, %t, %t, %t, %d, %t, %s)", mini, maxi, hasMini, hasMaxi, exMini, exMaxi, multiple, useMultiple, strings.Join(path.namedPath, "")))
		validationCode.Write(") {")
		validationCode.Indent()
		validationError(validationCode, "number check failed")
		validationCode.Dedent()
		validationCode.Write("}")
	}
	return !(hasMini || hasMaxi || useMultiple), nil
}

func generateString(ctx *Context, path *Path, desc *schemas.Type, writer *common.CodeWriter, globalCode *common.CodeWriter, validationCode *common.CodeWriter) (bool, error) {
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
	stringName := strings.Join(path.namedPath, "")
	if useMinLength || useMaxLength {
		validationCode.CommonLine()
		validationCode.Write("if (!")
		validationCode.Write(fmt.Sprintf("stringValidation(%d, %d, %t, %t, %s)", minLen, maxLen, useMinLength, useMaxLength, stringName))
		validationCode.Write(") {")
		validationCode.Indent()
		validationError(validationCode, "string check length failed")
		validationCode.Dedent()
		validationCode.Write("}")
	}
	if desc.Pattern != nil {
		validationCode.CommonLine()
		validationCode.Write("if (!")
		validationCode.Write(fmt.Sprintf("/%s/.test(%s)", *desc.Pattern, stringName))
		validationCode.Write(") {")
		validationCode.Indent()
		validationError(validationCode, "string check pattern failed")
		validationCode.Dedent()
		validationCode.Write("}")
	}
	return !(useMinLength || useMaxLength || desc.Pattern != nil), nil
}

func generateArray(ctx *Context, path *Path, desc *schemas.Type, writer *common.CodeWriter, globalCode *common.CodeWriter, validationCode *common.CodeWriter) (bool, error) {
	if desc.AdditionalItems != nil {
		return false, errors.New("only support single type array")
	}
	if desc.Items == nil {
		return false, errors.New("array must have item type")
	}
	arrayName := strings.Join(path.namedPath, "")
	validationCode.CommonLine()
	validationCode.Write(fmt.Sprintf("if (%s !== undefined) {", arrayName))
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
		validationCode.Write("if (!")
		validationCode.Write(fmt.Sprintf("arrayValidation(%d, %d, %t, %t, %t, %s)", mini, maxi, desc.MinItems != nil, desc.MaxItems != nil, desc.UniqueItems, arrayName))
		validationCode.Write(") {")
		validationCode.Indent()
		validationError(validationCode, "array check failed")
		validationCode.Dedent()
		validationCode.Write("}")
	}

	validationCode.Write(fmt.Sprintf("for (let item of %s) {", arrayName))
	validationCode.Indent()
	ignore, err := generateType(ctx, &Path{
		namedPath: []string{"item"},
		//quotePath: append(append([]string{}, path.quotePath...), "index"),
	}, desc.Items, writer, globalCode, validationCode)
	writer.Write("[]")
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

func generateObject(ctx *Context, path *Path, desc *schemas.Type, writer *common.CodeWriter, globalCode *common.CodeWriter, validationCode *common.CodeWriter) (bool, error) {
	writer.Write("{")
	writer.Indent()

	globalIgnore := true

	required := desc.Required
	if required == nil {
		required = []string{}
	}

	validationCode.CommonLine()

	sorted := sortKV{}
	for name, value := range desc.Properties {
		sorted = append(sorted, sortableKV{name, value})
	}
	sort.Sort(sorted)

	for _, iter := range sorted {
		name := iter.key
		value := iter.value.(*schemas.Type)
		propOptional := "?"
		for _, item := range required {
			if name == item {
				propOptional = ""
				break
			}
		}

		if propOptional == "" {
			validationCode.Write(fmt.Sprintf("if (%s.%s === undefiend) {", strings.Join(path.namedPath, ""), formatName(name)))
			validationCode.Indent()
			validationError(validationCode, "member cannot be undefined")
			validationCode.Dedent()
			validationCode.Write("}")
		}

		writer.CommonLine()
		writer.Write(fmt.Sprintf("\"%s\"%s: ", name, propOptional))
		ignore, err := generateType(ctx, &Path{
			namedPath: append(append([]string{}, path.namedPath...), "[\""+name+"\"]"),
			//quotePath: append(append([]string{}, path.quotePath...), "\""+name+"\""),
		}, value, writer, globalCode, validationCode)
		if err != nil {
			return false, err
		}

		globalIgnore = globalIgnore && ignore
	}

	writer.Dedent()
	writer.Write("}")
	return globalIgnore, nil
}

// ignore value & error
func generateType(ctx *Context, path *Path, desc *schemas.Type, writer *common.CodeWriter, globalCode *common.CodeWriter, validationCode *common.CodeWriter) (bool, error) {
	if desc == nil {
		return false, errors.New("must define type impl")
	}
	if desc.Ref != nil {
		parts := strings.Split(*desc.Ref, "/")
		if parts[0] != "#" {
			return false, errors.New("only local $ref is support")
		}
		parts = parts[1:]
		realName := []string{}
		for i, item := range parts {
			if i%2 != 0 {
				realName = append(realName, formatName(item))
			} else {
				if item != "$defs" && item != "definitions" {
					return false, errors.New("wrong $ref format")
				}
			}
		}
		writer.Write(strings.Join(realName, ""))

		validationCode.CommonLine()
		validationCode.Write(fmt.Sprintf("if ($checkTable[\"%s\"] !== undeifned) $checkTable[\"%s\"](%s);", strings.Join(realName, ""), strings.Join(realName, ""), strings.Join(path.namedPath, "")))

		return true, nil
	}
	if len(desc.Type) != 1 {
		// TODO: try union later by use interface{}
		return false, errors.New("multiple type is not supported")
	}
	// TODO: impl enum here
	switch desc.Type[0] {
	case schemas.TypeNameNull:
		ign, err := generateNull(ctx, path, desc, writer, globalCode, validationCode)
		return ign, err
	case schemas.TypeNameBoolean:
		ign, err := generateBoolean(ctx, path, desc, writer, globalCode, validationCode)
		return ign, err
	case schemas.TypeNameInteger:
		ign, err := generateInteger(ctx, path, desc, writer, globalCode, validationCode)
		return ign, err
	case schemas.TypeNameNumber:
		ign, err := generateNumber(ctx, path, desc, writer, globalCode, validationCode)
		return ign, err
	case schemas.TypeNameString:
		ign, err := generateString(ctx, path, desc, writer, globalCode, validationCode)
		return ign, err
	case schemas.TypeNameArray:
		ign, err := generateArray(ctx, path, desc, writer, globalCode, validationCode)
		return ign, err
	case schemas.TypeNameObject:
		ign, err := generateObject(ctx, path, desc, writer, globalCode, validationCode)
		return ign, err
	default:
		return false, errors.New(fmt.Sprintf("unknown type %s", desc.Type[0]))
	}
}

// TODO: from here

func GenerateCode(types map[string]*common.TypeDesc, config *TypescriptConfig, writer io.Writer) error {
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
		Tab:    "    ",
	}

	ctx := Context{}

	sortedType := sortKV{}
	for name, value := range types {
		sortedType = append(sortedType, sortableKV{name, value})
	}
	sort.Sort(sortedType)

	globalValidationBuffer := &bytes.Buffer{}
	globalValidationWriter := &common.CodeWriter{
		Writer: globalValidationBuffer,
		Tab:    "    ",
	}
	globalValidationWriter.CommonLine()
	globalValidationWriter.Write("const $checkTable: Record<string, (main: any) => void> = {")
	globalValidationWriter.Indent()

	fileWriter.CommonLine()
	fileWriter.Write("interface $typelist {")
	fileWriter.Indent()
	for _, iter := range sortedType {
		value := iter.value.(*common.TypeDesc)
		fileWriter.CommonLine()
		fileWriter.Write(fmt.Sprintf("%s: %s;", value.RenderedName, value.RenderedName))
	}
	fileWriter.Dedent()
	fileWriter.Write("}")
	fileWriter.CommonLine()
	fileWriter.CommonLine()
	fileWriter.Write("interface $typedChecker extends $typedCheckerImpl {")
	fileWriter.Indent()
	fileWriter.Write("check<K extends keyof $typelist>(type: K, main: $typelist[K]): void;")
	fileWriter.Dedent()
	fileWriter.Write("}")
	fileWriter.CommonLine()

	for _, iter := range sortedType {
		value := iter.value.(*common.TypeDesc)
		if value.Type.Enum != nil {
			if len(value.Type.Type) != 1 || value.Type.Type[0] != schemas.TypeNameString {
				return errors.New("only support string enum")
			}

			fileWriter.CommonLine()
			fileWriter.Write(fmt.Sprintf("export enum %s {", value.RenderedName))
			fileWriter.Indent()
			for _, item := range value.Type.Enum {
				if cased, ok := item.(string); !ok {
					return errors.New("only support string enum")
				} else {
					fileWriter.CommonLine()
					fileWriter.Write(fmt.Sprintf("%s = \"%s\",", formatName(cased), cased))
				}
			}
			fileWriter.Dedent()
			fileWriter.Write("}")
			fileWriter.CommonLine()

			globalValidationWriter.CommonLine()
			globalValidationWriter.Write(fmt.Sprintf("\"%s\": function (main?: %s) {", value.RenderedName, value.RenderedName))
			globalValidationWriter.Indent()
			globalValidationWriter.Write("if (main === undefined) return;")
			globalValidationWriter.CommonLine()
			globalValidationWriter.Write(fmt.Sprintf("if (!new Set<string>(Object.values(%s)).has(main)) {", value.RenderedName))
			globalValidationWriter.Indent()
			validationError(globalValidationWriter, "wrong enum value")
			globalValidationWriter.Dedent()
			globalValidationWriter.Write("}")
			globalValidationWriter.CommonLine()
			globalValidationWriter.Write("return;")
			globalValidationWriter.Dedent()
			globalValidationWriter.Write("},")
			continue
		}

		typeBuffer := &bytes.Buffer{}
		typeWriter := &common.CodeWriter{
			Writer: typeBuffer,
			Tab:    "    ",
		}
		validationBuffer := &bytes.Buffer{}
		validationWriter := &common.CodeWriter{
			Writer: validationBuffer,
			Tab:    "    ",
		}
		typeWriter.Write(fmt.Sprintf("export type %s = ", value.RenderedName))

		validationWriter.Indent()

		ignore, err := generateType(&ctx, &Path{
			namedPath: []string{"main"},
			//quotePath: []string{},
		}, value.Type, typeWriter, fileWriter, validationWriter)
		if err != nil {
			return err
		}

		fileWriter.CommonLine()
		fileWriter.Writer.Write(typeBuffer.Bytes())
		fileWriter.CommonLine()
		if !ignore {
			globalValidationWriter.CommonLine()
			globalValidationWriter.Write(fmt.Sprintf("\"%s\": function (main?: %s) {", value.RenderedName, value.RenderedName))
			globalValidationWriter.Indent()
			globalValidationWriter.Write("if (main === undefined) return;")

			globalValidationWriter.Writer.Write(validationBuffer.Bytes())

			globalValidationWriter.CommonLine()
			globalValidationWriter.Write("return;")
			globalValidationWriter.Dedent()
			globalValidationWriter.Write("},")
		}
	}

	globalValidationWriter.Dedent()
	globalValidationWriter.Write("}")
	globalValidationWriter.CommonLine()

	writer.Write(helperCode)
	writer.Write(fileBuffer.Bytes())
	writer.Write(globalValidationBuffer.Bytes())

	return nil
}
