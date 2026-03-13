package generator

import (
	"bytes"
	"text/template"

	"generator/extractor"
)

// Generator handles code generation from parsed service info
type Generator struct {
	wrapperTmpl  *template.Template
	registryTmpl *template.Template
}

// NewGenerator creates a new Generator instance
func NewGenerator() *Generator {
	return &Generator{
		wrapperTmpl:  template.Must(template.New("wrapper").Parse(wrapperTemplate)),
		registryTmpl: template.Must(template.New("registry").Parse(registryTemplate)),
	}
}

// GenerateWrapper generates wrapper code for all services
func (g *Generator) GenerateWrapper(services []*extractor.ServiceInfo) ([]byte, error) {
	var buf bytes.Buffer

	data := struct {
		Services []*extractor.ServiceInfo
		Package  string
	}{
		Services: services,
		Package:  "cache",
	}

	if err := g.wrapperTmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// GenerateRegistry generates registry code for all services
func (g *Generator) GenerateRegistry(services []*extractor.ServiceInfo) ([]byte, error) {
	var buf bytes.Buffer

	data := struct {
		Services []*extractor.ServiceInfo
		Package  string
	}{
		Services: services,
		Package:  "cache",
	}

	if err := g.registryTmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
