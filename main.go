package main

import (
	"./gioc"
	"fmt"
)

type A struct {
	B *B `di:""`
}

type B struct {
	name string
	age  int
	C    *C `di:"c"`
	A    *A `di:"a"`
}

type C struct {
	i    int
	b    bool
	name string
	A    *A `di:"a"`
}

func main() {
	opts := []gioc.Option{
		gioc.WithAllowEarlyReference(true),
		gioc.WithAllowPopulateStructBean(true),
	}
	ioc := gioc.NewIOC(opts...)

	// 这里在 Spring 中应该是由 Spring 扫描类路径然后获取 @Component 或者 @Import 注解的类的信息然后再注册的，我这里省去了扫描的过程，直接构建注册
	classA := gioc.NewClass("a", (*A)(nil), gioc.Singleton)
	classB := gioc.NewClass("bbbb", (*B)(nil), gioc.Singleton)
	classC := gioc.NewClass("c", (*C)(nil), gioc.Singleton)
	_ = ioc.Register(classA)
	_ = ioc.Register(classB)
	_ = ioc.Register(classC)
	bean := ioc.GetBean("a").(*A)
	bean2 := ioc.GetBean("a").(*A)
	fmt.Println(bean == bean2) // true

	// 即使 bbbb 是单例，但是由于不是 ptr 类型的，并且 golang 是值传递，所以这里返回的 bean 实际上已经不是 beanFactory 维护的那个 bean 了
	// 所以 IOC 实际上应该处理的是 ptr bean，非 ptr bean 处理没有意义
	//bean3 := ioc.GetBean("bbbb").(B)
	//fmt.Println(&bean3 == &(bean.B)) // false
	//bean3 := ioc.GetBean("bbbb").(*B)
	//fmt.Println(bean3 == bean.B) // true

	// C 是原型的，所以不会存储，所以这里会创建一个新的 C，因此跟单例 bean B 中的 C 不一样，输出 false
	// 如果将 C 改成 di:"s"，那么这里输出 true
	//bean4 := ioc.GetBean("C").(*C)
	//fmt.Println(bean4 == bean3.C) // false
}
