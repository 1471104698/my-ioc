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
	Register(class *Class) error
	// GetBean 根据 beanName 获取 bean
	GetBean(beanName string) interface{}
	// getSingleton 获取单例 bean（这里以后学习 Spring 建立三级缓存解决循环依赖）
	getSingleton(beanName string) interface{}
	// createBean 创建 bean 实例
	createBean(beanName string, beanType BeanType) interface{}
	// addSingleton 添加单例 bean
	addSingleton(beanName string, i interface{})
}

// AutowiredTag 变量注入注解
const AutowiredTag = "di"

// BeanNameTag 唯一标识 beanName 注解
const BeanNameTag = "beanName"

// BeanBeanFactory bean 工厂实现
type BeanBeanFactory struct {
	// 维护单例 bean 容器
	sc Container
	// 维护原型 bean 容器
	pc Container
	// 维护所有注册 bean 的类型
	btMap map[string]BeanType
	// 维护所有注册 bean 的类型信息
	tMap map[string]reflect.Type
	// 维护所有的单例 bean，一级缓存
	singletonMap map[string]interface{}
	// 维护早期暴露对象，用于解决循环依赖，二级缓存
	earlyMap map[string]interface{}
	// 工厂 map，三级缓存，用于 AOP bean
	factoryMap map[string]func() interface{}
	// 当前正在创建的 bean 列表
	creatingMap map[string]interface{}
	// 可选参数
	opts *Options
}

// NewBeanFactory 实例化一个 bean 工厂
func NewBeanFactory(opts ...Option) BeanFactory {
	bc := &BeanBeanFactory{
		btMap:        map[string]BeanType{},
		tMap:         map[string]reflect.Type{},
		singletonMap: map[string]interface{}{},
		earlyMap:     map[string]interface{}{},
		factoryMap:   map[string]func() interface{}{},
		creatingMap:  map[string]interface{}{},
		opts:         &Options{},
	}
	bc.sc = NewSingletonContainer(bc)
	bc.pc = NewPrototypeContainer(bc)
	if len(opts) > 0 {
		for _, opt := range opts {
			opt(bc.opts)
		}
	}
	return bc
}

// Register 注册一个 bean 到 beanFactory 中
func (bc *BeanBeanFactory) Register(class *Class) error {
	beanName := class.beanName
	beanType := class.beanType
	i := class.i
	if !isSingleton(beanType) && !isPrototype(beanType) {
		return fmt.Errorf("beanType: %v 不符合要求\n", beanType)
	}
	// 判断 beanName 是否已经注册过了，因为 beanName 是唯一标识，所以不能重复
	if bc.isRegistered(beanName) {
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
	// 处理 createBean 抛出的 panic
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()
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
func (bc *BeanBeanFactory) createBean(beanName string, beanType BeanType) interface{} {
	// 创建 bean 前置处理
	bc.createBefore(beanName, beanType)

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
	// 判断当前 beanName 对应的 reflect.Type 是否能够作为 bean
	if !isBean(t) {
		return nil
	}
	// 创建实例
	beanPtr := reflect.New(t)
	// 非 ptr bean value
	bean := beanPtr.Elem()

	// 判断是否允许暴露早期对象
	if bc.opts.allowEarlyReference {
		if t == tPtr {
			// 非 ptr bean
			bc.addSingletonFactory(beanName, bean.Interface())
		} else {
			// ptr bean
			bc.addSingletonFactory(beanName, beanPtr.Interface())
		}
	}
	// 扫描所有的 field
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// field 的 reflect.Type 类型信息
		ftPtr := field.Type
		// field 的 非 ptr type
		var ft reflect.Type
		if ftPtr.Kind() == reflect.Ptr {
			ft = ftPtr.Elem()
		} else {
			ft = ftPtr
		}
		// 非 bean，那么直接跳过
		if !isBean(ft) {
			continue
		}
		// 获取注入类型
		fieldBeanType := getFieldBeanType(field)
		// 不存在 autowire 注解，那么当前 field 不需要注入，那么跳过2
		if fieldBeanType == Invalid {
			continue
		}
		// 获取 field 对应注解的 beanName
		fieldBeanName := getFieldBeanName(bc, field, ft)
		// 判断是否需要注册到 beanFactory 中
		if !bc.isRegistered(fieldBeanName) {
			// 注册到 beanFactory 中
			_ = bc.Register(NewClass(fieldBeanName, ftPtr, fieldBeanType))
		}
		// 调用 GetBean() 获取 field bean，走 container 的逻辑
		fieldBean := bc.GetBean(fieldBeanName)
		// 获取不到 bean，那么跳过
		if fieldBean == nil {
			continue
		}
		// 将 bean 封装为 reflect.Value，用于 set()
		fieldBeanValue := reflect.ValueOf(fieldBean)
		// 将 field bean 赋值给 bean
		if ft == ftPtr {
			// field 非 ptr，那么直接设置即可
			bean.Field(i).Set(fieldBeanValue)
		} else {
			// field ptr，那么需要 fieldBean 是 ptr bean，这里需要先进行 Elem()，然后 Addr() 返回地址，赋值给 field
			bean.Field(i).Set(fieldBeanValue.Elem().Addr())
		}
	}
	// 创建 bean 后置处理
	bc.createAfter(beanName, beanType)

	// 返回非 ptr bean
	if t == tPtr {
		return bean.Interface()
	}
	// 返回 ptr bean
	return beanPtr.Interface()
}

// createBefore
func (bc *BeanBeanFactory) createBefore(beanName string, beanType BeanType) {
	// 原型 bean 直接返回
	if isPrototype(beanType) {
		return
	}
	// 判断当前 bean 是否正在创建
	if bc.creatingMap[beanName] != nil {
		panic(fmt.Errorf("bean %v is creating", beanName))
	}
	// 标识当前 bean 正在创建
	bc.creatingMap[beanName] = struct{}{}
}

// createAfter
func (bc *BeanBeanFactory) createAfter(beanName string, beanType BeanType) {
	// 原型 bean 直接返回
	if isPrototype(beanType) {
		return
	}
	// 将当前 bean 从正在创建 bean 列表中移除
	bc.creatingMap[beanName] = nil
}

// isSingleton 判断是否是单例 bean
func isSingleton(beanType BeanType) bool {
	return beanType == Singleton
}

// isPrototype 判断是否是原型 bean
func isPrototype(beanType BeanType) bool {
	return beanType == Prototype
}

// isRegistered 判断 beanName 是否已经注册
func (bc *BeanBeanFactory) isRegistered(beanName string) bool {
	_, exist := bc.tMap[beanName]
	return exist
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

// getSingleton 获取单例 bean（这里以后学习 Spring 建立三级缓存解决循环依赖）
func (bc *BeanBeanFactory) getSingleton(beanName string) interface{} {
	// 从单例池中获取
	bean := bc.singletonMap[beanName]
	// 单例池不存在 bean 并且允许循环依赖
	if bean == nil && bc.opts.allowEarlyReference {
		// 从早期暴露对象池中获取 bean
		bean = bc.earlyMap[beanName]
		if bean == nil {
			// 从三级缓存中获取
			singletonFactory := bc.factoryMap[beanName]
			if singletonFactory != nil {
				bean = singletonFactory()
				// 将 bean 放到早期对象池中，下次获取直接从早期对象池中获取
				bc.earlyMap[beanName] = bean
			}
		}
	}
	return bean
}

// addSingleton 添加单例 bean
func (bc *BeanBeanFactory) addSingleton(beanName string, bean interface{}) {
	bc.earlyMap[beanName] = nil
	bc.factoryMap[beanName] = nil
	bc.singletonMap[beanName] = bean
}

// addSingletonFactory
func (bc *BeanBeanFactory) addSingletonFactory(beanName string, bean interface{}) {
	// 设置工厂方法，这里主要是进行 AOP 处理
	bc.factoryMap[beanName] = func() interface{} {
		return bean
	}
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
	return field.Tag.Get(BeanNameTag)
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

// getFieldBeanName 获取字段变量的 beanName
func getFieldBeanName(bc *BeanBeanFactory, field reflect.StructField, ft reflect.Type) string {
	// 从 Tag 中尝试获取 beanName
	fieldBeanName := getBeanName(field)
	// 如果 field 没有对应的 beanName 注解，那么从注册的 bean 中找到相同类型的 bean 选择一个注入
	if fieldBeanName == "" {
		// 从已经注册的 bean 中尝试获取相同数据类型的 beanName
		fieldBeanName = bc.getBeanNameWithReflectType(ft)
		// 已注册的 bean 中不存在当前 field 类型，那么使用 ft.Name() 作为 beanName
		if fieldBeanName == "" {
			fieldBeanName = ft.Name()
		}
	}
	return fieldBeanName
}

// getFieldBeanType 获取变量注入类型
func getFieldBeanType(field reflect.StructField) BeanType {
	autowireTag := field.Tag.Get(AutowiredTag)
	if isSingleton(BeanType(autowireTag)) {
		return Singleton
	}
	if isPrototype(BeanType(autowireTag)) {
		return Prototype
	}
	return Invalid
}

// Option
type Option func(*Options)

// Options beanFactory 可选参数
type Options struct {
	// 是否允许暴露早期对象
	allowEarlyReference bool
}

// WithAllowEarlyReference
func WithAllowEarlyReference(allowEarlyReference bool) Option {
	return func(opts *Options) {
		opts.allowEarlyReference = allowEarlyReference
	}
}
