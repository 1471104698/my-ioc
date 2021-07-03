package gioc

// Bean 类型
type BeanType int

var (
	// 单例 bean
	Singleton BeanType = 1
	// 原型 bean
	Prototype BeanType = 2
)

// BeanFactory bean 工厂接口
type BeanFactory interface {
	// Register 注册一个 bean
	Register(i interface{})
	// GetBean 根据 beanName 获取 bean
	GetBean(beanName string) interface{}
}

// BeanBeanFactory bean 工厂实现
type BeanBeanFactory struct {
	// 维护单例 bean 容器
	sc *SingletonContainer
	// 维护原型 bean 容器
	pc *PrototypeContainer
	// 维护所有的 bean 类型信息
	ac container
}

// NewBeanFactory 实例化一个 bean 工厂
func NewBeanFactory() BeanFactory {
	return &BeanBeanFactory{
		sc: NewSingletonContainer(),
		pc: NewPrototypeContainer(),
		ac: container{},
	}
}

// Register 注册一个 bean
func (bc *BeanBeanFactory) Register(i interface{}) {}

// GetBean 根据 beanName 获取 bean 实例
func (bc *BeanBeanFactory) GetBean(beanName string) interface{} {
	return nil
}

// getBeanType 根据 beanName 获取 bean 类型
func (bc *BeanBeanFactory) getBeanType(beanName string) BeanType {
	return Singleton
}

// exist 根据 beanName 判断 bean 是否存在
func (bc *BeanBeanFactory) exist(beanName string) bool {
	return false
}
