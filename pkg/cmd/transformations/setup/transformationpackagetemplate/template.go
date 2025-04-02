package transformationpackagetemplate

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"
)

type PackageTemplate struct {
	TransformationName string
	TransformationID   string
	Sample             map[string]any
	Code               *string
}

type pkg struct {
	Name     string
	ID       string
	GlobalID string
}

type sample struct {
	Sample string
}

type code struct {
	Code *string
}

type helper struct{}

//go:embed code.tmpl
var codeTemplate string

//go:embed package.tmpl
var packageTemplate string

//go:embed sample.tmpl
var sampleTemplate string

//go:embed helper.tmpl
var helperTemplate string

func Generate(tmpl PackageTemplate) error {
	if err := execute(packageTemplate, "package.json", pkg{Name: tmpl.TransformationName, ID: tmpl.TransformationID}); err != nil {
		return err
	}

	if err := GenerateSample(tmpl.Sample); err != nil {
		return err
	}

	if err := execute(codeTemplate, "index.js", code{Code: tmpl.Code}); err != nil {
		return err
	}

	if err := execute(helperTemplate, "helper.ts", helper{}); err != nil {
		return err
	}

	return nil
}

func RefreshPackageJson(transformationName, transformationID string, globalTransformationID string) error {
	return execute(packageTemplate, "package.json", pkg{Name: transformationName, ID: transformationID, GlobalID: globalTransformationID})
}

func GenerateSample(sampleMap map[string]any) error {
	data, err := json.Marshal(sampleMap)
	if err != nil {
		return fmt.Errorf("unable to marshal sample data: %w", err)
	}

	return execute(sampleTemplate, "sample.js", sample{Sample: string(data)})
}

func execute(templateCode string, outputFile string, data any) error {
	tmpl, err := template.New(outputFile).Funcs(template.FuncMap{
		"hasPrefix": strings.HasPrefix,
	}).Parse(templateCode)
	if err != nil {
		return fmt.Errorf("unable to setup template for '%s' generator: %w", outputFile, err)
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
