package annotation

import (
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"go/ast"
)

type Parser interface {
	ParseSourceDir(dirName string, includeRegex string, excludeRegex string) (ParsedSources, error)
}

func NewParser() Parser {
	return &myParser{}
}

var debugAstOfSources = false

type myParser struct {
}

func (p *myParser) ParseSourceDir(dirName string, includeRegex string, excludeRegex string) (ParsedSources, error) {
	if debugAstOfSources {
		dumpFilesInDir(dirName)
	}
	packages, err := parseDir(dirName, includeRegex, excludeRegex)
	if err != nil {
		log.Printf("error parsing dir %s: %s", dirName, err.Error())
		return ParsedSources{}, err
	}

	v := &astVisitor{
		Imports: map[string]string{},
	}
	for _, aPackage := range packages {
		parsePackage(aPackage, v)
	}

	//embedOperationsInStructs(v)

	//embedTypedefDocLinesInEnum(v)

	return ParsedSources{
		Structs:    v.Structs,
		Operations: v.Operations,
		Interfaces: v.Interfaces,
		Typedefs:   v.Typedefs,
		Enums:      v.Enums,
	}, nil
}

func parsePackage(aPackage *ast.Package, v *astVisitor) {
	for _, fileEntry := range sortedFileEntries(aPackage.Files) {
		v.CurrentFilename = fileEntry.key

		appEngineOnly := true
		for _, commentGroup := range fileEntry.file.Comments {
			if commentGroup != nil {
				for _, comment := range commentGroup.List {
					if comment != nil && comment.Text == "// +build !appengine" {
						appEngineOnly = false
					}
				}
			}
		}
		if appEngineOnly {
			ast.Walk(v, &fileEntry.file)
		}
	}
}

func parseSourceFile(srcFilename string) (ParsedSources, error) {
	if debugAstOfSources {
		dumpFile(srcFilename)
	}

	v, err := doParseFile(srcFilename)
	if err != nil {
		log.Printf("error parsing src %s: %s", srcFilename, err.Error())
		return ParsedSources{}, err
	}

	return ParsedSources{
		Structs:    v.Structs,
		Operations: v.Operations,
		Interfaces: v.Interfaces,
		Typedefs:   v.Typedefs,
		Enums:      v.Enums,
	}, nil
}

func doParseFile(srcFilename string) (*astVisitor, error) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, srcFilename, nil, parser.ParseComments)
	if err != nil {
		log.Printf("error parsing src-file %s: %s", srcFilename, err.Error())
		return nil, err
	}
	v := &astVisitor{
		Imports: map[string]string{},
	}
	v.CurrentFilename = srcFilename
	ast.Walk(v, file)

	return v, nil
}

type fileEntry struct {
	key  string
	file ast.File
}

type fileEntries []fileEntry

func (list fileEntries) Len() int {
	return len(list)
}

func (list fileEntries) Less(i, j int) bool {
	return list[i].key < list[j].key
}

func (list fileEntries) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}

func sortedFileEntries(fileMap map[string]*ast.File) fileEntries {
	var fileEntries fileEntries = make([]fileEntry, 0, len(fileMap))
	for key, file := range fileMap {
		if file != nil {
			fileEntries = append(fileEntries, fileEntry{
				key:  key,
				file: *file,
			})
		}
	}
	sort.Sort(fileEntries)
	return fileEntries
}

func parseDir(dirName string, includeRegex string, excludeRegex string) (map[string]*ast.Package, error) {
	var includePattern = regexp.MustCompile(includeRegex)
	var excludePattern = regexp.MustCompile(excludeRegex)

	fileSet := token.NewFileSet()
	packageMap, err := parser.ParseDir(fileSet, dirName, func(fi os.FileInfo) bool {
		if excludePattern.MatchString(fi.Name()) {
			return false
		}
		return includePattern.MatchString(fi.Name())
	}, parser.ParseComments)
	if err != nil {
		log.Printf("error parsing dir %s: %s", dirName, err.Error())
		return packageMap, err
	}

	return packageMap, nil
}

func dumpFile(srcFilename string) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, srcFilename, nil, parser.ParseComments)
	if err != nil {
		log.Printf("error parsing src %s: %s", srcFilename, err.Error())
		return
	}
	ast.Print(fileSet, file)
}

func dumpFilesInDir(dirName string) {
	fileSet := token.NewFileSet()
	packageMap, err := parser.ParseDir(
		fileSet,
		dirName,
		nil,
		parser.ParseComments)
	if err != nil {
		log.Printf("error parsing dir %s: %s", dirName, err.Error())
	}
	for _, aPackage := range packageMap {
		for _, file := range aPackage.Files {
			ast.Print(fileSet, file)
		}
	}
}

// =====================================================================================================================

type astVisitor struct {
	CurrentFilename string
	PackageName     string
	Filename        string
	Imports         map[string]string
	Structs         []Struct
	Operations      []Operation
	Interfaces      []Interface
	Typedefs        []Typedef
	Enums           []Enum
}

func (v *astVisitor) Visit(node ast.Node) ast.Visitor {
	if node != nil {

		// package-name is in isolated node
		if packageName, ok := extractPackageName(node); ok {
			v.PackageName = packageName
		}

		// extract all imports into a map
		v.extractGenDeclImports(node)
		v.parseAsOperation(node)

	}
	return v
}

func (v *astVisitor) extractGenDeclImports(node ast.Node) {
	if genDecl, ok := node.(*ast.GenDecl); ok {
		for _, spec := range genDecl.Specs {
			if importSpec, ok := spec.(*ast.ImportSpec); ok {
				quotedImport := importSpec.Path.Value
				unquotedImport := strings.Trim(quotedImport, "\"")
				init, last := filepath.Split(unquotedImport)
				if init == "" {
					last = init
				}
				v.Imports[last] = unquotedImport
			}
		}
	}
}

func (v *astVisitor) parseAsOperation(node ast.Node) {
	// if mOperation, get its signature
	if mOperation := extractOperation(node, v.Imports); mOperation != nil {
		mOperation.PackageName = v.PackageName
		mOperation.Filename = v.CurrentFilename
		v.Operations = append(v.Operations, *mOperation)
	}
}

// =====================================================================================================================

func extractPackageName(node ast.Node) (string, bool) {
	if file, ok := node.(*ast.File); ok {
		if file.Name != nil {
			return file.Name.Name, true
		}
		return "", true
	}
	return "", false
}

// ------------------------------------------------------ STRUCT -------------------------------------------------------

func extractSpecsForStruct(specs []ast.Spec, imports map[string]string) *Struct {
	if len(specs) >= 1 {
		if typeSpec, ok := specs[0].(*ast.TypeSpec); ok {
			if structType, ok := typeSpec.Type.(*ast.StructType); ok {
				return &Struct{
					Name:   typeSpec.Name.Name,
					Fields: extractFieldList(structType.Fields, imports),
				}
			}
		}
	}
	return nil
}

func extractGenDeclForStruct(node ast.Node, imports map[string]string) *Struct {
	if genDecl, ok := node.(*ast.GenDecl); ok {
		// Continue parsing to see if it a struct
		if mStruct := extractSpecsForStruct(genDecl.Specs, imports); mStruct != nil {
			// Docline of struct (that could contain annotations) appear far before the details of the struct
			mStruct.DocLines = extractComments(genDecl.Doc)
			return mStruct
		}
	}
	return nil
}

func extractInterfaceMethods(fieldList *ast.FieldList, imports map[string]string) []Operation {
	methods := make([]Operation, 0)
	for _, field := range fieldList.List {
		if len(field.Names) > 0 {
			if funcType, ok := field.Type.(*ast.FuncType); ok {
				methods = append(methods, Operation{
					DocLines:   extractComments(field.Doc),
					Name:       field.Names[0].Name,
					InputArgs:  extractFieldList(funcType.Params, imports),
					OutputArgs: extractFieldList(funcType.Results, imports),
				})
			}
		}
	}
	return methods
}

// ----------------------------------------------------- OPERATION -----------------------------------------------------

func extractOperation(node ast.Node, imports map[string]string) *Operation {
	if funcDecl, ok := node.(*ast.FuncDecl); ok {
		mOperation := Operation{
			DocLines: extractComments(funcDecl.Doc),
		}

		if funcDecl.Recv != nil {
			fields := extractFieldList(funcDecl.Recv, imports)
			if len(fields) >= 1 {
				mOperation.RelatedStruct = &(fields[0])
			}
		}

		if funcDecl.Name != nil {
			mOperation.Name = funcDecl.Name.Name
		}

		if funcDecl.Type.Params != nil {
			mOperation.InputArgs = extractFieldList(funcDecl.Type.Params, imports)
		}

		if funcDecl.Type.Results != nil {
			mOperation.OutputArgs = extractFieldList(funcDecl.Type.Results, imports)
		}
		return &mOperation
	}
	return nil
}

func extractComments(commentGroup *ast.CommentGroup) []string {
	lines := make([]string, 0)
	if commentGroup != nil {
		for _, comment := range commentGroup.List {
			lines = append(lines, comment.Text)
		}
	}
	return lines
}

func extractTag(basicLit *ast.BasicLit) string {
	if basicLit != nil {
		return basicLit.Value
	}
	return ""
}

// ---------------------------------------------------------------------------------------------------------------------
