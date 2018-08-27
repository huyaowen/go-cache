package anno_t

import (
	"fmt"
)

type AnnoTest struct {
	*Aop
}

func NewAnnoTest() *AnnoTest {
	return &AnnoTest{}
}

// @cache(key="test")
func (a *AnnoTest) TestParse() {
	fmt.Println("test")
}
