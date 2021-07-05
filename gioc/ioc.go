package gioc

// Class 存储要注册的 bean 的信息
type Class struct {
	beanName string
	i        interface{}
	beanType BeanType
}

// NewClass
func NewClass(beanName string, i interface{}, beanType BeanType) *Class {
	return &Class{
		beanName: beanName,
		i:        i,
		beanType: beanType,
	}
}

// ioc 容器
type IOC struct {
	// beanFactory 维护一个 bean 工厂
	beanFactory BeanFactory
}

// NewIOC 实例化一个 IOC
func NewIOC(opts ...Option) *IOC {
	return &IOC{
		beanFactory: NewBeanFactory(opts...),
	}
}

// Register 调用 bean 工厂 注册一个 bean
func (ioc *IOC) Register(class *Class) error {
	return ioc.beanFactory.Register(class)
}

// GetBean 调用 bean 工厂 获取 bean
func (ioc *IOC) GetBean(beanName string) interface{} {
	return ioc.beanFactory.GetBean(beanName)
}
