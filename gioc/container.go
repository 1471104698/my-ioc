package gioc

// Container bean 容器接口
type Container interface {
	// Get 根据 beanName 获取 bean
	Get(beanName string) interface{}
}

// SingletonContainer 单例 bean 容器
type SingletonContainer struct {
	// 维护 beanFactory
	BeanFactory
}

// NewSingletonContainer 实例化一个单例 bean 容器
func NewSingletonContainer(beanFactory BeanFactory) Container {
	return &SingletonContainer{
		BeanFactory: beanFactory,
	}
}

// Get 获取 bean
func (sc *SingletonContainer) Get(beanName string) interface{} {
	// 先从缓存中获取
	bean := sc.getSingleton(beanName)
	if bean != nil {
		return bean
	}
	// 创建实例
	bean = sc.createBean(beanName, Singleton)
	if bean == nil {
		return nil
	}
	// 将 bean 添加到缓存中
	sc.addSingleton(beanName, bean)
	return bean
}

// PrototypeContainer 原型 bean 容器
type PrototypeContainer struct {
	// 维护 beanFactory
	BeanFactory
}

// NewPrototypeContainer 实例化一个原型 bean 容器
func NewPrototypeContainer(beanFactory BeanFactory) Container {
	return &PrototypeContainer{
		BeanFactory: beanFactory,
	}
}

// Get 获取 bean
func (pc *PrototypeContainer) Get(beanName string) interface{} {
	// 创建实例
	bean := pc.createBean(beanName, Prototype)
	if bean == nil {
		return nil
	}
	// 原型 bean 不需要添加到缓存中
	return bean
}
