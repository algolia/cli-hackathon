package transformationpackagetemplate

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"
)

type PackageTemplate struct {
	TransformationName string
	Sample             map[string]any
	Code               *string
}

type pkg struct {
	Name string
}

type sample struct {
	Sample string
}

type code struct {
	Code *string
}

type helper struct{}

func Generate(tmpl PackageTemplate) error {
	if err := tmpl.execute("package.tmpl", "package.json", pkg{Name: tmpl.TransformationName}); err != nil {
		return err
	}

	data, err := json.Marshal(tmpl.Sample)
	if err != nil {
		return fmt.Errorf("unable to marshal sample data: %w", err)
	}

	if err := tmpl.execute("sample.tmpl", "sample.js", sample{Sample: string(data)}); err != nil {
		return err
	}

	if err := tmpl.execute("code.tmpl", "index.js", code{Code: tmpl.Code}); err != nil {
		return err
	}

	if err := tmpl.execute("helper.tmpl", "helper.ts", helper{}); err != nil {
		return err
	}

	return nil
}

func (t PackageTemplate) execute(templateFile string, outputFile string, data any) error {
	tmpl, err := template.New(templateFile).Funcs(template.FuncMap{
		"hasPrefix": strings.HasPrefix,
	}).ParseFiles("pkg/cmd/transformations/setup/transformationpackagetemplate/" + templateFile)
	if err != nil {
		return fmt.Errorf("unable to setup template for '%s' generator: %w", templateFile, err)
	}

	file, err := os.Create(outputFile)
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
