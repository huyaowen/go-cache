package spel

import (
	"reflect"
	"strings"
	"time"
)

// EvaluationContext 求值上下文
type EvaluationContext struct {
	Args map[string]interface{}
	ArgValues []interface{}
	ArgNames []string
	Result interface{}
	Target interface{}
	TargetType reflect.Type
	Method, CacheName, Key string
	TTL time.Duration
	Extra map[string]interface{}
	Timestamp time.Time
}

func NewEvaluationContext() *EvaluationContext {
	return &EvaluationContext{
		Args: make(map[string]interface{}),
		ArgValues: make([]interface{}, 0),
		ArgNames: make([]string, 0),
		Extra: make(map[string]interface{}),
		Timestamp: time.Now(),
	}
}

func (c *EvaluationContext) SetArg(name string, v interface{}) {
	if c.Args == nil { c.Args = make(map[string]interface{}) }
	c.Args[name] = v
}

func (c *EvaluationContext) SetArgByIndex(i int, v interface{}) {
	for len(c.ArgValues) <= i { c.ArgValues = append(c.ArgValues, nil) }
	c.ArgValues[i] = v
}

func (c *EvaluationContext) SetArgName(i int, name string) {
	for len(c.ArgNames) <= i { c.ArgNames = append(c.ArgNames, "") }
	c.ArgNames[i] = name
}

func (c *EvaluationContext) GetArg(name string) (interface{}, bool) {
	if c.Args == nil { return nil, false }
	v, ok := c.Args[name]
	return v, ok
}

func (c *EvaluationContext) GetArgByIndex(i int) (interface{}, bool) {
	if i < 0 || i >= len(c.ArgValues) { return nil, false }
	return c.ArgValues[i], true
}

func (c *EvaluationContext) SetResult(v interface{}) { c.Result = v }

func (c *EvaluationContext) SetExtra(k string, v interface{}) {
	if c.Extra == nil { c.Extra = make(map[string]interface{}) }
	c.Extra[k] = v
}

func (c *EvaluationContext) GetExtra(k string) (interface{}, bool) {
	if c.Extra == nil { return nil, false }
	v, ok := c.Extra[k]
	return v, ok
}

func (c *EvaluationContext) BuildVariables() map[string]interface{} {
	vars := make(map[string]interface{})
	// Add named args (p0, p1, id, user, etc.) and also support #name syntax
	for n, v := range c.Args {
		vars[n] = v
		// Support #name syntax for named args (e.g., #id, #user)
		if !strings.HasPrefix(n, "#") {
			vars["#"+n] = v
		}
	}
	// Add indexed args (#0, #1, etc.) and also p0, p1 for compatibility
	for i, v := range c.ArgValues {
		idxStr := itoa(i)
		vars["#"+idxStr] = v
		vars["p"+idxStr] = v
	}
	if c.Result != nil { vars["result"] = c.Result; vars["returnValue"] = c.Result }
	if c.Target != nil { vars["target"] = c.Target; vars["this"] = c.Target }
	if c.Method != "" { vars["method"] = c.Method }
	if c.CacheName != "" { vars["cacheName"] = c.CacheName }
	if c.Key != "" { vars["key"] = c.Key }
	vars["timestamp"] = c.Timestamp.Unix()
	vars["now"] = c.Timestamp
	return vars
}

func itoa(i int) string {
	if i == 0 { return "0" }
	neg := false
	if i < 0 { neg = true; i = -i }
	var d []byte
	for i > 0 { d = append([]byte{byte('0'+i%10)}, d...); i /= 10 }
	if neg { d = append([]byte{'-'}, d...) }
	return string(d)
}
