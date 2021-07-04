package main

import (
	"./gioc"
	"fmt"
)

type A struct {
	B B `di:"s" beanName:"bbbb"`
}

type B struct {
	name string
	age  int
	C    *C `di:"p"`
}

type C struct {
	i    int
	b    bool
	name string
}

func main() {
	ioc := gioc.NewIOC()
	err := ioc.Register("a", (*A)(nil), gioc.Singleton)
	if err != nil {
		fmt.Println(err)
	}
	bean := ioc.GetBean("a").(*A)
	fmt.Println(bean.B)
	fmt.Println(*(bean.B.C))
}
