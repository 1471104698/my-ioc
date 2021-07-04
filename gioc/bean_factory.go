package gioc

import (
	"fmt"
	"reflect"
)

// Bean 类型
type BeanType string

var (
	// 无效类型，bean 没有注册或者没有标注正确的 bean 类型
	Invalid BeanType = ""
	// 单例 bean
	Singleton BeanType = "singleton"
	// 原型 bean
	Prototype BeanType = "prototype"
)

// BeanFactory bean 工厂接口
type BeanFactory interface {
	// Register 注册一个 bean
	Register(beanName string, i interface{}, beanType BeanType) error
	// GetBean 根据 beanName 获取 bean
	GetBean(beanName string) interface{}
	// getSingleton 获取单例 bean（这里以后学习 Spring 建立三级缓存解决循环依赖）
	getSingleton(beanName string) interface{}
}

// AutowireTag 变量注入注解
const AutowireTag = "autowire"

// BeanName 唯一标识 bean 注解
const BeanName = "beanName"

// BeanBeanFactory bean 工厂实现
type BeanBeanFactory struct {
	// 维护单例 bean 容器
	sc *SingletonContainer
	// 维护原型 bean 容器
	pc *PrototypeContainer
	// 维护所有注册 bean 的类型
	btMap map[string]BeanType
	// 维护所有注册 bean 的类型信息
	tMap map[string]reflect.Type
	// 维护所有的单例 bean
	beanMap map[string]interface{}
}

// NewBeanFactory 实例化一个 bean 工厂
func NewBeanFactory() BeanFactory {
	bc := &BeanBeanFactory{
		btMap:   map[string]BeanType{},
		tMap:    map[string]reflect.Type{},
		beanMap: map[string]interface{}{},
	}
	bc.sc = NewSingletonContainer(bc)
	bc.pc = NewPrototypeContainer(bc)
	return bc
}

// Register 注册一个 bean 到 beanFactory 中
func (bc *BeanBeanFactory) Register(beanName string, i interface{}, beanType BeanType) error {
	if !isSingleton(beanType) && !isPrototype(beanType) {
		return fmt.Errorf("beanType: %v 不符合要求\n", beanType)
	}
	// 判断 beanName 是否已经注册过了，因为 beanName 是唯一标识，所以不能重复
	if _, exist := bc.btMap[beanName]; exist {
		return fmt.Errorf("beanName was registered by other bean")
	}
	bc.btMap[beanName] = beanType
	bc.tMap[beanName] = reflect.TypeOf(i).Elem()
	return nil
}

// GetBean 根据 beanName 获取 bean 实例
func (bc *BeanBeanFactory) GetBean(beanName string) interface{} {
	// 获取 bean 类型
	beanType := bc.getBeanType(beanName)
	// bean 不存在
	if beanType == Invalid {
		return nil
	}
	var bean interface{}
	if isSingleton(beanType) {
		bean = bc.sc.Get(beanName)
	} else {
		bean = bc.pc.Get(beanName)
	}
	return bean
}

// createBean 创建 bean 实例
func (bc *BeanBeanFactory) createBean(beanName string, beanType BeanType, t reflect.Type) interface{} {
	if !isBean(t) || (!isSingleton(beanType) && !isPrototype(beanType)) {
		return nil
	}
	// 创建实例
	bean := reflect.New(t)
	// 扫描所有的 field
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		ft := field.Type
		if !isBean(ft) {
			continue
		}
		// 获取注入类型
		autowireType := getAutowireType(field)
		if autowireType == Invalid {
			continue
		}
		// 获取 beanName
		fieldBeanName := getBeanName(field)
		// 如果 field 没有对应的 beanName 注解，那么从注册的 bean 中找到相同类型的 bean 选择一个注入
		if fieldBeanName == "" {

		}
		// 获取 field bean
		fieldBean := bc.GetBean(fieldBeanName)
		if fieldBean == nil {
			continue
		}
		// 将 field bean 设置到 bean 中
		bean.Field(i).Set(reflect.ValueOf(fieldBean))
	}
	return false
}

// getBeanType 根据 beanName 获取 bean 类型
func (bc *BeanBeanFactory) getBeanType(beanName string) BeanType {
	beanType, exist := bc.btMap[beanName]
	if !exist {
		return Invalid
	}
	return beanType
}

// getBeanName 获取 field 注解的 beanName，作为 IOC 容器中唯一 bean 标识
func getBeanName(field reflect.StructField) string {
	return field.Tag.Get(BeanName)
}

// isSingleton 判断是否是单例 bean
func isSingleton(beanType BeanType) bool {
	return beanType == Singleton
}

// isPrototype 判断是否是原型 bean
func isPrototype(beanType BeanType) bool {
	return beanType == Prototype
}

// getSingleton 获取单例 bean（这里以后学习 Spring 建立三级缓存解决循环依赖）
func (bc *BeanBeanFactory) getSingleton(beanName string) interface{} {
	bean, exist := bc.beanMap[beanName]
	if exist {
		return bean
	}
	return nil
}

// isBean 判断是否能够作为 bean，基本数据类型等不能作为一个 bean
func isBean(t reflect.Type) bool {
	t = t.Elem()
	// reflect.Interface 是 reflect.TypeOf(&i).Elem().Kind() 指针传入然后调用 Elem() 返回的类型，因为 reflect 没有具体确定它的类型
	// 这里判断有点问题，因为 var i int 传入 &i 那么这里得到的也是 Interface，无法做更加具体的区分
	if t.Kind() == reflect.Struct || t.Kind() == reflect.Interface {
		return true
	}
	return false
}

// getAutowireType 获取变量注入类型
func getAutowireType(field reflect.StructField) BeanType {
	autowireTag := field.Tag.Get(AutowireTag)
	if isSingleton(BeanType(autowireTag)) {
		return Singleton
	}
	if isPrototype(BeanType(autowireTag)) {
		return Prototype
	}
	return Invalid
}
