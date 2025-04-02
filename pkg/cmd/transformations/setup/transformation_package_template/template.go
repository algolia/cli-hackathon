package transformation_package_template

import (
	"encoding/json"
	"fmt"
	"os"
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

type sample struct {
	Sample string
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

	if err := tmpl.execute("code.tmpl", "index.js", map[string]any{}); err != nil {
		return err
	}

	if err := tmpl.execute("helper.tmpl", "helper.ts", helper{}); err != nil {
		return err
	}

	return nil
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
