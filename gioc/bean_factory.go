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
	Singleton BeanType = "s"
	// 原型 bean
	Prototype BeanType = "p"
)

// BeanFactory bean 工厂接口
type BeanFactory interface {
	// Register 注册一个 bean
	Register(beanName string, i interface{}, beanType BeanType) error
	// GetBean 根据 beanName 获取 bean
	GetBean(beanName string) interface{}
	// getSingleton 获取单例 bean（这里以后学习 Spring 建立三级缓存解决循环依赖）
	getSingleton(beanName string) interface{}
	// createBean 创建 bean 实例
	createBean(beanName string) interface{}
	// addSingleton 添加单例 bean
	addSingleton(beanName string, i interface{})
}

// AutowireTag 变量注入注解
const AutowireTag = "di"

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
	var t reflect.Type
	t, ok := i.(reflect.Type)
	if !ok {
		// 这里不调用 Elem()，因为可能注册的就是一个指针类型，因此这里不做指针处理
		t = reflect.TypeOf(i)
	}
	bc.btMap[beanName] = beanType
	bc.tMap[beanName] = t
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
func (bc *BeanBeanFactory) createBean(beanName string) interface{} {
	// 获取 bean 类型信息
	tPtr, exist := bc.tMap[beanName]
	if !exist {
		return nil
	}
	// 非 ptr type
	var t reflect.Type
	if tPtr.Kind() == reflect.Ptr {
		t = tPtr.Elem()
	} else {
		t = tPtr
	}
	if !isBean(t) {
		return nil
	}
	// 创建实例
	beanPtr := reflect.New(t)
	bean := beanPtr.Elem()

	// 扫描所有的 field
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		ftPtr := field.Type
		var ft reflect.Type
		if ftPtr.Kind() == reflect.Ptr {
			ft = ftPtr.Elem()
		} else {
			ft = ftPtr
		}
		// 非 bean 返回
		if !isBean(ft) {
			continue
		}
		// 获取注入类型
		autowireType := getAutowireType(field)
		// 不存在 autowire 注解，那么当前 field 不需要注入，那么跳过
		if autowireType == Invalid {
			continue
		}
		// 获取 field 对应注解的 beanName
		fieldBeanName := getBeanName(field)
		// 如果 field 没有对应的 beanName 注解，那么从注册的 bean 中找到相同类型的 bean 选择一个注入
		if fieldBeanName == "" {
			// 从已经注册的 bean 中尝试获取相同数据类型的 beanName
			fieldBeanName = bc.getBeanNameWithReflectType(ft)
			// 不存在，那么使用 t.Name() 作为 beanName
			if fieldBeanName == "" {
				fieldBeanName = ft.Name()
				// 注册到 beanFactory 中
				_ = bc.Register(fieldBeanName, ftPtr, autowireType)
			}
		}
		// 调用 GetBean() 获取 field bean，走 container 的逻辑
		fieldBean := bc.GetBean(fieldBeanName)
		if fieldBean == nil {
			continue
		}
		fieldBeanValue := reflect.ValueOf(fieldBean)
		// 将 field bean 设置到 bean 中
		if ft == ftPtr {
			// field 非 ptr，那么直接设置即可
			bean.Field(i).Set(fieldBeanValue)
		} else {
			// field ptr，那么需要 fieldBean 是 ptr bean，这里需要先进行 Elem()，然后 Addr() 返回地址，赋值给 field
			bean.Field(i).Set(fieldBeanValue.Elem().Addr())
		}
	}
	// 返回非 ptr bean
	if t == tPtr {
		return bean.Interface()
	}
	// 返回 ptr bean
	return beanPtr.Interface()
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
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// reflect.Interface 是 reflect.TypeOf(&i).Elem().Kind() 指针传入然后调用 Elem() 返回的类型，因为 reflect 没有具体确定它的类型
	// 这里判断有点问题，因为 var i int 传入 &i 那么这里得到的也是 Interface，无法做更加具体的区分
	if t.Kind() == reflect.Struct || t.Kind() == reflect.Interface {
		return true
	}
	return false
}

// addSingleton 添加单例 bean
func (bc *BeanBeanFactory) addSingleton(beanName string, i interface{}) {
	bc.beanMap[beanName] = i
}

// getBeanType 根据 beanName 获取 bean 类型
func (bc *BeanBeanFactory) getBeanType(beanName string) BeanType {
	beanType, exist := bc.btMap[beanName]
	if !exist {
		return Invalid
	}
	return beanType
}

// getBeanNameWithReflectType 根据 reflect.Type 从已经注册的 bean 中获取对应的 beanName
func (bc *BeanBeanFactory) getBeanNameWithReflectType(tape reflect.Type) string {
	// 这里操作次数并不多，因此不需要特地维护一个 map，直接从原有 map 扫描获取即可，单纯的时间换空间
	for beanName, t := range bc.tMap {
		if t == tape {
			return beanName
		}
	}
	return ""
}

// getBeanName 获取 field 注解的 beanName，作为 IOC 容器中唯一 bean 标识
func getBeanName(field reflect.StructField) string {
	return field.Tag.Get(BeanName)
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
