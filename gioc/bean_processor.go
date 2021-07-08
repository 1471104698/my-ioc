package gioc

import (
	"reflect"
)

// BeanProcessor bean 处理器（Spring BeanPostProcessor bean 后置处理器简化版）
type BeanProcessor interface {
	// processPropertyValues 属性注入
	processPropertyValues(wrapBean reflect.Value, t reflect.Type)
	// processBeforeInstantiation bean 初始化前处理函数，用户可以在这里自定义 bean 的创建逻辑
	// 如果返回 bean != nil，那么不会再执行 createBean
	processBeforeInstantiation(beanName string, t reflect.Type) interface{}
	// postProcessAfterInitialization bean 初始化后处理函数，也是 AOP 的处理逻辑
	processAfterInitialization(beanName string, bean interface{}, t reflect.Type) interface{}
}

// PopulateBeanProcessor field 填充 bean 处理器
type PopulateBeanProcessor struct {
	bc *BeanBeanFactory
}

// NewPopulateBeanProcessor
func NewPopulateBeanProcessor(bc *BeanBeanFactory) BeanProcessor {
	return &PopulateBeanProcessor{
		bc: bc,
	}
}

// processPropertyValues 属性注入
func (bp *PopulateBeanProcessor) processPropertyValues(wrapBean reflect.Value, t reflect.Type) {
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
			// 不允许非 ptr 结构体注入
			if !bp.bc.isAllowPopulateStructBean() {
				continue
			}
			ft = ftPtr
		}
		// 非 wrapBean，那么直接跳过
		if !isBean(ft) {
			continue
		}
		// 获取注入类型
		fieldBeanType := getFieldBeanType(field)
		// 不存在 di 注解，那么当前 field 不需要注入，那么跳过2
		if fieldBeanType == Invalid {
			continue
		}
		// 获取 field 对应注解的 beanName
		fieldBeanName := getFieldBeanName(bp.bc, field, ft)
		// 判断是否需要注册到 beanFactory 中
		if !bp.bc.isRegistered(fieldBeanName) {
			// 注册到 beanFactory 中
			_ = bp.bc.Register(NewClass(fieldBeanName, ftPtr, fieldBeanType))
		}
		// 调用 GetBean() 获取 field wrapBean，走 container 的逻辑
		fieldBean := bp.bc.GetBean(fieldBeanName)
		// 获取不到 wrapBean，那么跳过
		if fieldBean == nil {
			continue
		}
		// 将 wrapBean 封装为 reflect.Value，用于 set()
		fieldBeanValue := reflect.ValueOf(fieldBean)
		// 将 field wrapBean 赋值给 wrapBean
		if ft == ftPtr {
			// field 非 ptr，那么直接设置即可
			wrapBean.Field(i).Set(fieldBeanValue)
		} else {
			// field ptr，那么需要 fieldBean 是 ptr wrapBean，这里需要先进行 Elem()，然后 Addr() 返回地址，赋值给 field
			wrapBean.Field(i).Set(fieldBeanValue.Elem().Addr())
		}
	}
}

// processBeforeInstantiation
func (bp *PopulateBeanProcessor) processBeforeInstantiation(beanName string, t reflect.Type) interface{} {
	return nil
}

// processAfterInitialization
func (bp *PopulateBeanProcessor) processAfterInitialization(beanName string, bean interface{}, t reflect.Type) interface{} {
	return nil
}

// AopBeanProcessor aop bean 处理器
type AopBeanProcessor struct {
	bc *BeanBeanFactory
	// 存储早期对象 AOP 处理过的 beanName 列表
	earlyProxyReferences map[string]interface{}
}

// NewAopBeanProcessor
func NewAopBeanProcessor(bc *BeanBeanFactory) BeanProcessor {
	return &AopBeanProcessor{
		bc:                   bc,
		earlyProxyReferences: map[string]interface{}{},
	}
}

// processPropertyValues
func (bp *AopBeanProcessor) processPropertyValues(wrapBean reflect.Value, t reflect.Type) {
}

// processBeforeInstantiation
func (bp *AopBeanProcessor) processBeforeInstantiation(beanName string, t reflect.Type) interface{} {
	return nil
}

// processAfterInitialization
func (bp *AopBeanProcessor) processAfterInitialization(beanName string, bean interface{}, t reflect.Type) interface{} {
	// 作为早期对象的时候已经处理过了
	if bp.earlyProxyReferences[beanName] != nil {
		return bean
	}
	return bp.wrapIfNecessary(beanName, bean)
}

// wrapIfNecessary AOP 处理
func (bp *AopBeanProcessor) wrapIfNecessary(beanName string, bean interface{}) interface{} {
	bp.earlyProxyReferences[beanName] = struct{}{}
	return bean
}
