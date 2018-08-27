package anno_t

import (
	"fmt"

	"github.com/gogap/aop"
)

type Aop struct{}

// use join point to around the real func of login
func (p *Aop) Around(pjp aop.ProceedingJoinPointer) {
	fmt.Println("@Begin Around")

	fmt.Println("@End Around")
}
