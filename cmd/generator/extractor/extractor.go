package extractor

import (
	"regexp"
)

// Annotation represents a cache annotation (supports @cacheable, @cacheput, @cacheevict)
type Annotation struct {
	Type      string // Method name
	CacheName string // Cache name
	Key       string // Cache key expression
	TTL       string // Time to live
	Condition string // Condition expression (SpEL)
	Unless    string // Unless expression (SpEL)
	Before    bool   // Execute before method (for cacheevict)
	Sync      bool   // Synchronous execution
	AnnotationType string // Annotation type: cacheable, cacheput, cacheevict
}

// ParamInfo represents a method parameter
type ParamInfo struct {
	Name string
	Type string
}

// ResultInfo represents a method return value
type ResultInfo struct {
	Name  string
	Type  string
	Index int
}

// MethodInfo represents a method with its annotation
type MethodInfo struct {
	Name       string
	Annotation *Annotation
	Params     []*ParamInfo
	Results    []*ResultInfo
	HasError   bool
}

// ServiceInfo represents a service struct with its methods
type ServiceInfo struct {
	Name       string            // Generated decorator name
	ImplType   string            // Original struct type name
	Package    string            // Package name
	ImportPath string            // Import path
	Imports    map[string]string // Required imports (alias -> path)
	Methods    []*MethodInfo
}

// ExtractAnnotation extracts annotation info from a comment string
// Supports: @cacheable, @cacheput, @cacheevict
// Example: // @cacheable(cache="users", key="#id", ttl="30m")
func ExtractAnnotation(comment string) *Annotation {
	// Match @cacheable, @cacheput, @cacheevict annotations
	re := regexp.MustCompile(`//\s*@(cacheable|cacheput|cacheevict)\s*\((.*)\)`)
	matches := re.FindStringSubmatch(comment)
	
	if len(matches) != 3 {
		return nil
	}
	
	annotationType := matches[1]
	paramsStr := matches[2]
	params := parseAnnotationParams(paramsStr)
	
	if len(params) == 0 {
		return nil
	}
	
	annotation := &Annotation{
		AnnotationType: annotationType,
		CacheName:      getParam(params, "cache"),
		Key:            getParam(params, "key"),
		TTL:            getParam(params, "ttl"),
		Condition:      getParam(params, "condition"),
		Unless:         getParam(params, "unless"),
		Before:         getParam(params, "before") == "true",
		Sync:           getParam(params, "sync") == "true",
	}
	
	// Validate required fields
	if annotation.CacheName == "" {
		return nil
	}
	
	return annotation
}

// getParam gets a parameter value from the params map
func getParam(params map[string]string, key string) string {
	if val, ok := params[key]; ok {
		return val
	}
	return ""
}

// parseAnnotationParams parses the parameters from an annotation string
func parseAnnotationParams(paramsStr string) map[string]string {
	params := make(map[string]string)
	
	// Match key="value" patterns
	re := regexp.MustCompile(`(\w+)\s*=\s*"([^"]*)"`)
	matches := re.FindAllStringSubmatch(paramsStr, -1)
	
	for _, match := range matches {
		if len(match) == 3 {
			params[match[1]] = match[2]
		}
	}
	
	return params
}
