package parser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"generator/extractor"
)

// Parser handles AST parsing of Go source files
type Parser struct {
	fset *token.FileSet
}

// NewParser creates a new Parser instance
func NewParser() *Parser {
	return &Parser{
		fset: token.NewFileSet(),
	}
}

// ParseFile parses a Go source file and returns the AST
func (p *Parser) ParseFile(filename string) (*ast.File, error) {
	return parser.ParseFile(p.fset, filename, nil, parser.ParseComments)
}

// ExtractAnnotations extracts @cacheable annotations from the AST
func (p *Parser) ExtractAnnotations(file *ast.File) []*extractor.Annotation {
	var annotations []*extractor.Annotation

	// Walk through all declarations
	for _, decl := range file.Decls {
		// Check function declarations
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Doc != nil {
				for _, comment := range fn.Doc.List {
					annotation := extractor.ExtractAnnotation(comment.Text)
					if annotation != nil {
						annotation.Type = fn.Name.Name
						annotations = append(annotations, annotation)
					}
				}
			}
		}
	}

	return annotations
}

// ExtractServices extracts service information from the AST
func (p *Parser) ExtractServices(file *ast.File, annotations []*extractor.Annotation) []*extractor.ServiceInfo {
	var services []*extractor.ServiceInfo

	// Build annotation map: method name -> annotation
	annotationMap := make(map[string]*extractor.Annotation)
	for _, ann := range annotations {
		annotationMap[ann.Type] = ann
	}

	// Find struct types and their methods
	typeStructMap := make(map[string]*ast.StructType)
	typeNameMap := make(map[string]string) // receiver type name -> struct name

	// First pass: collect struct types
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					if structType, ok := typeSpec.Type.(*ast.StructType); ok {
						typeStructMap[typeSpec.Name.Name] = structType
					}
				}
			}
		}
	}

	// Second pass: collect methods and group by receiver
	methodMap := make(map[string][]*extractor.MethodInfo)

	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			// Check if it's a method (has receiver)
			if fn.Recv != nil && len(fn.Recv.List) > 0 {
				receiver := fn.Recv.List[0]
				
				// Get receiver type name
				var receiverName string
				switch t := receiver.Type.(type) {
				case *ast.Ident:
					receiverName = t.Name
				case *ast.StarExpr:
					if ident, ok := t.X.(*ast.Ident); ok {
						receiverName = ident.Name
					}
				}

				if receiverName == "" {
					continue
				}

				// Check if this method has an annotation
				ann, hasAnnotation := annotationMap[fn.Name.Name]
				if !hasAnnotation {
					continue
				}

				// Extract method info
				methodInfo := p.extractMethodInfo(fn, ann)
				methodMap[receiverName] = append(methodMap[receiverName], methodInfo)
				typeNameMap[receiverName] = receiverName
			}
		}
	}

	// Build service info
	for typeName, methods := range methodMap {
		if len(methods) == 0 {
			continue
		}

		serviceInfo := &extractor.ServiceInfo{
			Name:     typeName + "Decorator",
			ImplType: typeName,
			Methods:  methods,
		}

		services = append(services, serviceInfo)
	}

	return services
}

// extractMethodInfo extracts method information from a function declaration
func (p *Parser) extractMethodInfo(fn *ast.FuncDecl, annotation *extractor.Annotation) *extractor.MethodInfo {
	methodInfo := &extractor.MethodInfo{
		Name:        fn.Name.Name,
		Annotation:  annotation,
		Params:      p.extractParams(fn.Type.Params),
		Results:     p.extractResults(fn.Type.Results),
		HasError:    p.hasErrorResult(fn.Type.Results),
	}

	return methodInfo
}

// extractParams extracts parameter information
func (p *Parser) extractParams(fields *ast.FieldList) []*extractor.ParamInfo {
	if fields == nil {
		return nil
	}

	var params []*extractor.ParamInfo
	for _, field := range fields.List {
		for _, name := range field.Names {
			paramInfo := &extractor.ParamInfo{
				Name: name.Name,
				Type: p.typeToString(field.Type),
			}
			params = append(params, paramInfo)
		}
		// Handle unnamed parameters
		if len(field.Names) == 0 {
			paramInfo := &extractor.ParamInfo{
				Name: "_",
				Type: p.typeToString(field.Type),
			}
			params = append(params, paramInfo)
		}
	}

	return params
}

// extractResults extracts result information
func (p *Parser) extractResults(fields *ast.FieldList) []*extractor.ResultInfo {
	if fields == nil {
		return nil
	}

	var results []*extractor.ResultInfo
	for i, field := range fields.List {
		for _, name := range field.Names {
			resultInfo := &extractor.ResultInfo{
				Name:  name.Name,
				Type:  p.typeToString(field.Type),
				Index: i,
			}
			results = append(results, resultInfo)
		}
		// Handle unnamed results
		if len(field.Names) == 0 {
			resultInfo := &extractor.ResultInfo{
				Name:  "",
				Type:  p.typeToString(field.Type),
				Index: i,
			}
			results = append(results, resultInfo)
		}
	}

	return results
}

// hasErrorResult checks if the function returns an error
func (p *Parser) hasErrorResult(fields *ast.FieldList) bool {
	if fields == nil {
		return false
	}

	for _, field := range fields.List {
		if p.typeToString(field.Type) == "error" {
			return true
		}
	}

	return false
}

// typeToString converts an AST type to a string representation
func (p *Parser) typeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + p.typeToString(t.X)
	case *ast.SelectorExpr:
		return p.typeToString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + p.typeToString(t.Elt)
	case *ast.MapType:
		return "map[" + p.typeToString(t.Key) + "]" + p.typeToString(t.Value)
	case *ast.ChanType:
		return "chan " + p.typeToString(t.Value)
	case *ast.FuncType:
		return p.funcTypeToString(t)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct{}"
	default:
		return "interface{}"
	}
}

// funcTypeToString converts a function type to string
func (p *Parser) funcTypeToString(ft *ast.FuncType) string {
	params := p.fieldListToString(ft.Params)
	results := p.fieldListToString(ft.Results)

	if results == "" {
		return "func(" + params + ")"
	}
	return "func(" + params + ") " + results
}

// fieldListToString converts a field list to string
func (p *Parser) fieldListToString(fields *ast.FieldList) string {
	if fields == nil {
		return ""
	}

	var parts []string
	for _, field := range fields.List {
		typeStr := p.typeToString(field.Type)
		if len(field.Names) > 0 {
			var names []string
			for _, name := range field.Names {
				names = append(names, name.Name)
			}
			parts = append(parts, strings.Join(names, ", ")+" "+typeStr)
		} else {
			parts = append(parts, typeStr)
		}
	}

	return strings.Join(parts, ", ")
}
