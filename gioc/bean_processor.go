package gioc

import (
	"fmt"
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
			// 不允许非 ptr 结构体注入，那么直接跳过该 field
			if !bp.bc.isAllowPopulateStructBean() {
				continue
			}
			ft = ftPtr
		}
		// 非 bean，那么直接跳过
		if !isBean(ft) {
			continue
		}
		// 获取 field 注入的 bean 的 beanName
		fieldBeanName := bp.getFieldBeanName(field, ftPtr, ft)
		// beanName 不存在，报错
		if fieldBeanName == "" {
			panic(fmt.Errorf("field bean %v is not exist", field.Name))
		}
		// 调用 GetBean() 获取 field bean，走 container 的逻辑
		var fieldBean interface{}
		// 如果是 struct bean，那么不使用旧的 bean，直接获取一个新的 bean，因为即使使用
		if isStructBean(ftPtr, ft) {
			fieldBean = bp.bc.GetNewBean(fieldBeanName)
		} else {
			fieldBean = bp.bc.GetBean(fieldBeanName)
		}
		// 将 field bean 封装为 reflect.Value，用于 set
		fieldBeanValue := reflect.ValueOf(fieldBean)
		if fieldBeanValue.Kind() == reflect.Ptr {
			fieldBeanValue = fieldBeanValue.Elem()
		}
		// 将 field bean 赋值给 wrapBean
		if isStructBean(ftPtr, ft) {
			// field 非 ptr，那么直接设置即可
			wrapBean.Field(i).Set(fieldBeanValue)
		} else {
			// field ptr，那么需要 fieldBean 是 ptr bean，这里需要先进行 Elem()，然后 Addr() 返回地址，赋值给 field
			wrapBean.Field(i).Set(fieldBeanValue.Addr())
		}
	}
}

// isStructBean 判断是否是 struct bean（非 ptr）
func isStructBean(ftPtr, ft reflect.Type) bool {
	return ftPtr == ft
}

// getFieldBeanName 获取字段变量的 beanName
func (bp *PopulateBeanProcessor) getFieldBeanName(field reflect.StructField, ftPtr, ft reflect.Type) string {
	fieldBeanName, exist := hasAndGetDI(field)
	// 不需要注入
	if !exist {
		return ""
	}
	// 存在 di 注解，但是没有指定注入的 bean 的 beanName，那么从注册的 bean 中找到相同类型的 bean，选择一个注入
	if fieldBeanName == "" {
		// 从已经注册的 bean 中尝试获取相同数据类型的 beanName
		fieldBeanName = bp.bc.getBeanNameWithReflectType([]reflect.Type{ftPtr, ft})
	}
	// 判断 fieldBeanName 是否存在
	if _, exist = bp.bc.tMap[fieldBeanName]; !exist {
		return ""
	}
	return fieldBeanName
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
