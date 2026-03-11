package spel

import (
	"fmt"
	"reflect"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
)

// SpELEvaluator 表达式求值器
type SpELEvaluator struct {
	cache map[string]*vm.Program
}

// SpELEvaluatorInterface 接口
type SpELEvaluatorInterface interface {
	Evaluate(exprStr string, ctx *EvaluationContext) (interface{}, error)
	EvaluateToString(exprStr string, ctx *EvaluationContext) (string, error)
	EvaluateToInt(exprStr string, ctx *EvaluationContext) (int64, error)
	ClearCache()
	CacheSize() int
}

// NewSpELEvaluator 创建求值器
func NewSpELEvaluator() *SpELEvaluator {
	return &SpELEvaluator{cache: make(map[string]*vm.Program)}
}

// Evaluate 求值表达式
func (e *SpELEvaluator) Evaluate(exprStr string, ctx *EvaluationContext) (interface{}, error) {
	if exprStr == "" {
		return nil, ErrEmptyExpression
	}

	program, err := e.compile(exprStr)
	if err != nil {
		return nil, err
	}

	variables := ctx.BuildVariables()
	result, err := expr.Run(program, variables)
	if err != nil {
		return nil, &EvaluationError{Expression: exprStr, Err: err}
	}

	return result, nil
}

// EvaluateToString 求值为字符串
func (e *SpELEvaluator) EvaluateToString(exprStr string, ctx *EvaluationContext) (string, error) {
	result, err := e.Evaluate(exprStr, ctx)
	if err != nil {
		return "", err
	}
	if result == nil {
		return "", nil
	}
	if s, ok := result.(string); ok {
		return s, nil
	}
	return fmt.Sprintf("%v", result), nil
}

// EvaluateToInt 求值为整数
func (e *SpELEvaluator) EvaluateToInt(exprStr string, ctx *EvaluationContext) (int64, error) {
	result, err := e.Evaluate(exprStr, ctx)
	if err != nil {
		return 0, err
	}
	return toInt64(result)
}

func (e *SpELEvaluator) compile(exprStr string) (*vm.Program, error) {
	if program, ok := e.cache[exprStr]; ok {
		return program, nil
	}

	program, err := expr.Compile(exprStr, expr.AllowUndefinedVariables())
	if err != nil {
		return nil, &CompilationError{Expression: exprStr, Err: err}
	}

	e.cache[exprStr] = program
	return program, nil
}

// ClearCache 清除缓存
func (e *SpELEvaluator) ClearCache() {
	e.cache = make(map[string]*vm.Program)
}

// CacheSize 获取缓存大小
func (e *SpELEvaluator) CacheSize() int {
	return len(e.cache)
}

func toInt64(v interface{}) (int64, error) {
	switch val := v.(type) {
	case int:
		return int64(val), nil
	case int8:
		return int64(val), nil
	case int16:
		return int64(val), nil
	case int32:
		return int64(val), nil
	case int64:
		return val, nil
	case uint:
		return int64(val), nil
	case uint8:
		return int64(val), nil
	case uint16:
		return int64(val), nil
	case uint32:
		return int64(val), nil
	case uint64:
		return int64(val), nil
	case float32:
		return int64(val), nil
	case float64:
		return int64(val), nil
	case string:
		var r int64
		_, err := fmt.Sscanf(val, "%d", &r)
		return r, err
	default:
		rv := reflect.ValueOf(v)
		if rv.CanInt() {
			return rv.Int(), nil
		}
		return 0, fmt.Errorf("cannot convert %T to int64", v)
	}
}

// CompilationError 编译错误
type CompilationError struct {
	Expression string
	Err        error
}

func (e *CompilationError) Error() string {
	return fmt.Sprintf("compile '%s': %v", e.Expression, e.Err)
}

func (e *CompilationError) Unwrap() error {
	return e.Err
}

// EvaluationError 求值错误
type EvaluationError struct {
	Expression string
	Err        error
}

func (e *EvaluationError) Error() string {
	return fmt.Sprintf("eval '%s': %v", e.Expression, e.Err)
}

func (e *EvaluationError) Unwrap() error {
	return e.Err
}

// ErrEmptyExpression 空表达式错误
var ErrEmptyExpression = fmt.Errorf("expression cannot be empty")

var _ SpELEvaluatorInterface = (*SpELEvaluator)(nil)
