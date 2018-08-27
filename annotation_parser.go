package main

import (
	"encoding/json"
	"fmt"
	"go-cache/anno_t"
	"go-cache/annotation"

	"github.com/gogap/aop"
)

const (
	GenfilePrefix       = "gen_"
	GenfileExcludeRegex = GenfilePrefix + ".*"
)

func main() {
	parser := annotation.NewParser()
	sources, _ := parser.ParseSourceDir("./anno_t", "^.*.go$", GenfileExcludeRegex)
	fmt.Println(sources)
	b, _ := json.Marshal(sources)
	fmt.Println(string(b))
	fmt.Println(sources.Operations[1].DocLines)
	registry := annotation.NewRegistry([]annotation.AnnotationDescriptor{
		{
			Name:       "cache",
			ParamNames: []string{},
			Validator:  validateFunc,
		},
	})
	a, _ := registry.ResolveAnnotationByName(sources.Operations[1].DocLines, "cache")
	data, _ := json.Marshal(a)
	fmt.Println(string(data))

	fmt.Println("=====================================================")

	beanFactory := aop.NewClassicBeanFactory()
	beanFactory.RegisterBean("test", new(anno_t.AnnoTest))
	//beanFactory.RegisterBean("foo", new(Foo))

	aspect := aop.NewAspect("aspect_1", "test")
	aspect.SetBeanFactory(beanFactory)

	pointcut := aop.NewPointcut("pointcut_1").Execution(`TestParse()`)
	pointcut.Execution(`TestParse()`)

	aspect.AddPointcut(pointcut)

	aspect.AddAdvice(&aop.Advice{Ordering: aop.Around, Method: "Around", PointcutRefID: "pointcut_1"})

	gogapAop := aop.NewAOP()
	gogapAop.SetBeanFactory(beanFactory)
	gogapAop.AddAspect(aspect)

	anno_t.NewAnnoTest().TestParse()

	var err error
	var proxy *aop.Proxy

	// Get proxy
	if proxy, err = gogapAop.GetProxy("test"); err != nil {
		fmt.Println("get proxy failed", err)
		return
	}

	proxy.Invoke(new(anno_t.AnnoTest).TestParse)
}

func validateFunc(annot annotation.Annotation) bool {
	return true
}
