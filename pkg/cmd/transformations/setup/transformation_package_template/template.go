package transformation_package_template

import (
	"fmt"
	"os"
	"text/template"
)

type pkgTemplate struct {
	outputDirectory    string
	transformationName string
}

type pkg struct {
	Name string
}

type code struct{}

func Generate(outputDirectory, transformationName string) error {
	tmpl := pkgTemplate{outputDirectory: outputDirectory, transformationName: transformationName}

	if err := tmpl.pkg(); err != nil {
		return err
	}

	if err := tmpl.code(); err != nil {
		return err
	}

	return nil
}

func (t pkgTemplate) pkg() error {
	return t.execute("package.tmpl", "package.json", pkg{Name: t.transformationName})
}

func (t pkgTemplate) code() error {
	return t.execute("code.tmpl", "index.js", code{})
}

func (t pkgTemplate) execute(templateFile string, outputFile string, data any) error {
	tmpl, err := template.New(templateFile).ParseFiles("pkg/cmd/transformations/setup/transformation_package_template/" + templateFile)
	if err != nil {
		return fmt.Errorf("unable to setup template for '%s' generator: %w", templateFile, err)
	}

	file, err := os.Create(fmt.Sprintf("%s%c%s", t.outputDirectory, os.PathSeparator, outputFile))
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
