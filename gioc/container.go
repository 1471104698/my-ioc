package gioc

import "reflect"

// Container bean 容器接口
type Container interface {
	// Add 添加 bean，如果已经存在，那么不重复添加
	Add(i interface{})
	// Get 根据 beanName 获取 bean
	Get(beanName string) interface{}
}

// container 存储 bean 类型信息
type container map[string]reflect.Type

// SingletonContainer 单例 bean 容器
type SingletonContainer struct {
	// 存储 bean 类型信息
	c container
	// 存储 bean 实例
	beans map[string]interface{}
}

// NewSingletonContainer 实例化一个单例 bean 容器
func NewSingletonContainer() *SingletonContainer {
	return &SingletonContainer{
		c:     container{},
		beans: map[string]interface{}{},
	}
}

// Add
func (sc *SingletonContainer) Add(i interface{}) {}

// Get
func (sc *SingletonContainer) Get(beanName string) interface{} {
	return nil
}

// PrototypeContainer 原型 bean 容器
type PrototypeContainer container

// NewPrototypeContainer 实例化一个原型 bean 容器
func NewPrototypeContainer() *PrototypeContainer {
	return &PrototypeContainer{}
}

// Add
func (pc *PrototypeContainer) Add(i interface{}) {}

// Get
func (pc *PrototypeContainer) Get(beanName string) interface{} {
	return nil
}
