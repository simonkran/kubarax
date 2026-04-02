package templates

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

//go:embed embedded
var embeddedFS embed.FS

// TemplateType defines the category of templates to process
type TemplateType int

const (
	TemplateTypeAll       TemplateType = iota
	TemplateTypeHelm
	TemplateTypeTerraform
)

const (
	DefaultManagedCatalogPath  = "managed-service-catalog"
	DefaultCustomerCatalogPath = "customer-service-catalog"
)

// TemplateResult holds the output path and rendered content for a processed template
type TemplateResult struct {
	Path    string
	Content []byte
}

// GetEmbeddedTemplatesList returns all template file paths matching the given type
func GetEmbeddedTemplatesList(templateType TemplateType) ([]string, error) {
	var paths []string

	err := fs.WalkDir(embeddedFS, "embedded", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		// Filter by template type
		switch templateType {
		case TemplateTypeHelm:
			if !strings.Contains(path, "/helm/") {
				return nil
			}
		case TemplateTypeTerraform:
			if !strings.Contains(path, "/terraform/") {
				return nil
			}
		}

		paths = append(paths, path)
		return nil
	})

	return paths, err
}

// TemplateFiles processes a list of template files with the given data context
func TemplateFiles(paths []string, data map[string]interface{}) ([]TemplateResult, error) {
	var results []TemplateResult
	var errs []error

	for _, path := range paths {
		content, err := embeddedFS.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Errorf("reading %s: %w", path, err))
			continue
		}

		// Process .tplt files through Go template engine
		if strings.HasSuffix(path, ".tplt") {
			rendered, err := renderTemplate(path, string(content), data)
			if err != nil {
				errs = append(errs, fmt.Errorf("templating %s: %w", path, err))
				continue
			}
			content = []byte(rendered)
		}

		results = append(results, TemplateResult{
			Path:    path,
			Content: content,
		})
	}

	return results, errors.Join(errs...)
}

// TemplateAllFiles combines GetEmbeddedTemplatesList and TemplateFiles
func TemplateAllFiles(templateType TemplateType, data map[string]interface{}) ([]TemplateResult, error) {
	paths, err := GetEmbeddedTemplatesList(templateType)
	if err != nil {
		return nil, fmt.Errorf("listing templates: %w", err)
	}

	return TemplateFiles(paths, data)
}

// renderTemplate processes a single template string with sprig functions
func renderTemplate(name, content string, data map[string]interface{}) (string, error) {
	funcMap := sprig.TxtFuncMap()

	tmpl, err := template.New(filepath.Base(name)).Funcs(funcMap).Parse(content)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return buf.String(), nil
}
