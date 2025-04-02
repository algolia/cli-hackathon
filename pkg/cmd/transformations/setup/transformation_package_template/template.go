package transformation_package_template

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"text/template"
)

type PackageTemplate struct {
	OutputDirectory    string
	TransformationName string
	Sample             map[string]any
}

type pkg struct {
	Name string
}

type code struct {
	Typedef string
}

type helper struct{}

func Generate(tmpl PackageTemplate) error {
	if err := tmpl.execute("package.tmpl", "package.json", pkg{Name: tmpl.TransformationName}); err != nil {
		return err
	}

	if err := tmpl.execute("code.tmpl", "index.ts", code{Typedef: tmpl.generateTypedefFromSample()}); err != nil {
		return err
	}

	if err := tmpl.execute("helper.tmpl", "helper.ts", helper{}); err != nil {
		return err
	}

	if err := tmpl.execute("tsconfig.tmpl", "tsconfig.json", helper{}); err != nil {
		return err
	}

	if err := tmpl.execute("code.tmpl", "index.ts", code{}); err != nil {
		return err
	}

	sample, err := json.Marshal(tmpl.Sample)
	if err != nil {
		return fmt.Errorf("unable to marshal sample data: %w", err)
	}

	if err := os.WriteFile(fmt.Sprintf("%s%c%s", tmpl.OutputDirectory, os.PathSeparator, "sample.json"), sample, 0o750); err != nil {
		return fmt.Errorf("unable to write to 'sample.json' file: %w", err)
	}

	return nil
}

func (t PackageTemplate) generateTypedefFromSample() string {
	var sb strings.Builder

	sb.WriteString("/**\n * @typedef {Object} SourceRecord\n")

	for key, value := range t.Sample {
		jsType := ""

		switch reflect.TypeOf(value).Kind() {
		case reflect.String:
			jsType = "string"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			jsType = "number"
		case reflect.Bool:
			jsType = "boolean"
		case reflect.Slice, reflect.Array:
			jsType = "Array"
		case reflect.Map, reflect.Struct:
			jsType = "Object"
		default:
			jsType = "any"
		}

		sb.WriteString(fmt.Sprintf(" * @property {%s} %s\n", jsType, key))
	}

	sb.WriteString(" */\n")

	return sb.String()
}

func (t PackageTemplate) execute(templateFile string, outputFile string, data any) error {
	tmpl, err := template.New(templateFile).ParseFiles("pkg/cmd/transformations/setup/transformation_package_template/" + templateFile)
	if err != nil {
		return fmt.Errorf("unable to setup template for '%s' generator: %w", templateFile, err)
	}

	file, err := os.Create(fmt.Sprintf("%s%c%s", t.OutputDirectory, os.PathSeparator, outputFile))
	if err != nil {
		return fmt.Errorf("unable to open '%s' file: %w", outputFile, err)
	}
	defer file.Close()

	err = tmpl.Execute(file, data)
	if err != nil {
		return fmt.Errorf("unable to execute template for '%s' file: %w", outputFile, err)
	}

	return nil
}
