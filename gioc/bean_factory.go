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
	// RegisterBeanProcessor 注册 bean 处理器
	RegisterBeanProcessor(class *Class) error
	// GetBean 根据 beanName 获取 bean
	GetBean(beanName string) interface{}
	// getSingleton 获取单例 bean（这里以后学习 Spring 建立三级缓存解决循环依赖）
	getSingleton(beanName string, allowEarlyReference bool) interface{}
	// createBean 创建 bean 实例
	createBean(beanName string, beanType BeanType) interface{}
	// addSingleton 添加单例 bean
	addSingleton(beanName string, i interface{})
	// isAllowEarlyReference 是否允许循环依赖
	isAllowEarlyReference() bool
}

// DITag 变量注入注解
const DITag = "di"

// initBeanProcessors 初始bean 处理器列表
var initBeanProcessors = []func(*BeanBeanFactory) BeanProcessor{
	NewPopulateBeanProcessor,
	NewAopBeanProcessor,
}

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
	// bean 处理器集合
	beanProcessors []BeanProcessor
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
	for _, bp := range initBeanProcessors {
		bc.beanProcessors = append(bc.beanProcessors, bp(bc))
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

func (bc *BeanBeanFactory) RegisterBeanFunc() {

}

// RegisterBeanProcessor 注册 bean 处理器
func (bc *BeanBeanFactory) RegisterBeanProcessor(class *Class) error {
	class.beanType = Singleton
	err := bc.Register(class)
	if err != nil {
		return err
	}
	bpBean := bc.GetBean(class.beanName)
	bp, ok := bpBean.(BeanProcessor)
	if !ok {
		bc.tMap = nil
		bc.singletonMap = nil
		delete(bc.btMap, class.beanName)
		return fmt.Errorf("bean %v is not a bean processor", class.beanName)
	}
	bc.beanProcessors = append(bc.beanProcessors, bp)
	return nil
}

// GetBean 根据 beanName 获取 bean 实例
func (bc *BeanBeanFactory) GetBean(beanName string) interface{} {
	// 处理 createBean 抛出的 panic
	//defer func() {
	//	if err := recover(); err != nil {
	//		fmt.Println(err)
	//	}
	//}()
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
	// bean 创建的前置处理
	bc.createBefore(beanName, beanType)
	// bean 创建完毕的后置处理
	defer bc.createAfter(beanName, beanType)

	// 获取 bean 类型信息
	t, exist := bc.tMap[beanName]
	if !exist {
		return nil
	}
	// 创建 bean 前看该 bean 是否存在特殊创建逻辑
	bean := bc.resolveBeforeInstantiation(beanName, t)
	if bean != nil {
		return bean
	}
	// 创建 bean
	return bc.doCreateBean(beanName, t)
}

// doCreateBean 真正的创建 bean 实例逻辑
func (bc *BeanBeanFactory) doCreateBean(beanName string, tPtr reflect.Type) interface{} {
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
			bc.addSingletonFactory(beanName, bean.Interface(), t)
		} else {
			// ptr bean
			bc.addSingletonFactory(beanName, beanPtr.Interface(), t)
		}
	}
	// 属性注入
	bc.populateBean(bean, t)

	// 初始化 bean，这里会执行 AOP 处理
	// 注意这里需要传入 ptr bean，为了跟下面的 getSingleton 对齐
	bean2 := bc.initializeBean(beanName, beanPtr.Interface(), t)

	// 上面存在两种 bean，一种是原始的 bean1，一种是 initializeBean 初始化返回的 bean2
	// 创建 A bean 的时候有以下几种情况：
	// 	1、创建 A 的时候 A 作为早期对象暴露了，那么如果 A 依赖了 B，B 依赖了 A，那么 A 会被拿出放到 earlyMap 中
	//	  如果 A 需要 AOP 的话，那么 AOP 对象就在 earlyMap 中，那么 initializeBean 返回的就是原始 bean
	// 	2、创建 A 的时候 A 不作为早期对象暴露，或者没有构成循环依赖，那么 initializeBean 中返回的可能是 AOP bean
	// 综上，我们实际上需要再获取 earlyMap 中的 bean3，bean2 和 bean3 之间具有以下关系：
	// 	1、如果 A 没有暴露早期对象或者没有循环依赖，那么 bean2 就是最终需要返回的 bean
	// 	2、如果 A 存在循环依赖，那么 bean3 就是最终需要返回的 bean
	var resBean interface{}
	// 允许循环依赖
	if bc.isAllowEarlyReference() {
		// 判断是否出现了循环依赖
		// 这里的 resBean 就是上面讲的 bean3
		resBean = bc.getSingleton(beanName, false)
		// 为空，那么没有出现循环依赖，那么最终 bean 为 bean2
		if resBean == nil {
			resBean = bean2
		}
	} else {
		// 不允许循环依赖，那么最终 bean2 为 bean2
		resBean = bean2
	}
	// 返回非 ptr bean
	if t == tPtr {
		// 如果 resBean 是 ptr，所以这里借助 reflect.Value 返回非 ptr
		resBeanV := reflect.ValueOf(resBean)
		if resBeanV.Kind() == reflect.Ptr {
			return resBeanV.Elem().Interface()
		} else {
			return resBean
		}
	}
	// 返回 ptr bean
	return resBean
}

// resolveBeforeInstantiation 初始化 bean 前的处理
func (bc *BeanBeanFactory) resolveBeforeInstantiation(beanName string, t reflect.Type) interface{} {
	var bean interface{}
	for _, bp := range bc.beanProcessors {
		bean = bp.processBeforeInstantiation(beanName, t)
		if bean != nil {
			return bean
		}
	}
	return nil
}

// populateBean 属性注入
func (bc *BeanBeanFactory) populateBean(bean reflect.Value, t reflect.Type) {
	for _, bp := range bc.beanProcessors {
		bp.processPropertyValues(bean, t)
	}
}

// initializeBean 创建完 bean 后初始化 bean
func (bc *BeanBeanFactory) initializeBean(beanName string, bean interface{}, t reflect.Type) interface{} {
	wrapBean := bean
	for _, bp := range bc.beanProcessors {
		bean = bp.processAfterInitialization(beanName, wrapBean, t)
		if bean != nil {
			return bean
		}
	}
	return bean
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
func (bc *BeanBeanFactory) getSingleton(beanName string, allowEarlyReference bool) interface{} {
	// 从单例池中获取
	bean := bc.singletonMap[beanName]
	// 单例池不存在 bean 并且允许循环依赖
	if bean == nil {
		// 从早期暴露对象池中获取 bean
		bean = bc.earlyMap[beanName]
		if bean == nil && allowEarlyReference {
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
func (bc *BeanBeanFactory) addSingletonFactory(beanName string, bean interface{}, t reflect.Type) {
	// 设置工厂方法，这里主要是进行 AOP 处理
	bc.factoryMap[beanName] = func() interface{} {
		// 注意这里是闭包的，后面修改了 bean 所以这里需要对 bean 进行一份备份
		wrapBean := bean
		for _, bp := range bc.beanProcessors {
			bean = bp.processAfterInitialization(beanName, wrapBean, t)
			if bean != nil {
				return bean
			}
		}
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

// getBeanNameWithReflectType 根据 reflect.Type 从已经注册的 bean 中获取对应的 beanName
func (bc *BeanBeanFactory) getBeanNameWithReflectType(types []reflect.Type) string {
	// 这里操作次数并不多，因此不需要特地维护一个 map，直接从原有 map 扫描获取即可，单纯的时间换空间
	for beanName, t := range bc.tMap {
		for _, ft := range types {
			if t == ft {
				return beanName
			}
		}
	}
	return ""
}

// getFieldBeanType 获取变量注入类型
func getFieldBeanType(field reflect.StructField) BeanType {
	autowireTag := field.Tag.Get(DITag)
	if isSingleton(BeanType(autowireTag)) {
		return Singleton
	}
	if isPrototype(BeanType(autowireTag)) {
		return Prototype
	}
	return Invalid
}

// hasAndGetDI 判断是否存在 DI 注解，同时获取 DI 注解的值
func hasAndGetDI(field reflect.StructField) (string, bool) {
	return field.Tag.Lookup(DITag)
}

// isAllowEarlyReference 是否允许循环依赖
func (bc *BeanBeanFactory) isAllowEarlyReference() bool {
	return bc.opts.allowEarlyReference
}

// isAllowPopulateStructBean 是否允许注入非 ptr bean
func (bc *BeanBeanFactory) isAllowPopulateStructBean() bool {
	return bc.opts.allowPopulateStructBean
}

// Option
type Option func(*Options)

// Options beanFactory 可选参数
type Options struct {
	// 是否允许暴露早期对象
	allowEarlyReference bool
	// 是否允许注入非 ptr bean
	allowPopulateStructBean bool
}

// WithAllowEarlyReference
func WithAllowEarlyReference(allowEarlyReference bool) Option {
	return func(opts *Options) {
		opts.allowEarlyReference = allowEarlyReference
	}
}

// WithAllowPopulateStructBean
func WithAllowPopulateStructBean(allowPopulateStructBean bool) Option {
	return func(opts *Options) {
		opts.allowPopulateStructBean = allowPopulateStructBean
	}
}
