package gioc

// ioc 容器
type IOC struct {
	// beanFactory 维护一个 bean 工厂
	beanFactory BeanFactory
}

// NewIOC 实例化一个 IOC
func NewIOC() *IOC {
	return &IOC{
		beanFactory: NewBeanFactory(),
	}
}

// Register 调用 bean 工厂 注册一个 bean
func (ioc *IOC) Register(beanName string, i interface{}, beanType BeanType) error {
	return ioc.beanFactory.Register(beanName, i, beanType)
}

// GetBean 调用 bean 工厂 获取 bean
func (ioc *IOC) GetBean(beanName string) interface{} {
	return ioc.beanFactory.GetBean(beanName)
}
