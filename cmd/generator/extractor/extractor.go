package extractor

import (
	"regexp"
)

// Annotation represents a @cacheable annotation
type Annotation struct {
	Type      string // Method name
	CacheName string // Cache name
	Key       string // Cache key expression
	TTL       string // Time to live
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
	Name     string       // Generated decorator name
	ImplType string       // Original struct type name
	Methods  []*MethodInfo
}

// ExtractAnnotation extracts annotation info from a comment string
// Example: // @cacheable(cache="users", key="#id", ttl="30m")
func ExtractAnnotation(comment string) *Annotation {
	// Match @cacheable annotation
	re := regexp.MustCompile(`//\s*@cacheable\s*\((.*)\)`)
	matches := re.FindStringSubmatch(comment)
	
	if len(matches) != 2 {
		return nil
	}
	
	paramsStr := matches[1]
	params := parseAnnotationParams(paramsStr)
	
	if len(params) == 0 {
		return nil
	}
	
	annotation := &Annotation{
		CacheName: getParam(params, "cache"),
		Key:       getParam(params, "key"),
		TTL:       getParam(params, "ttl"),
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
