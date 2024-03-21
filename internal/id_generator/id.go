package idgenerator

import (
	"sync"
)

type IdGenerator struct {
	i    int32
	lock sync.Mutex
}

func (id *IdGenerator) Get() int32 {
	id.lock.Lock()
	defer id.lock.Unlock()
	id.i = id.i + 1
	if id.i == 128 {
		id.i = 1
	}
	return id.i
}

var instance *IdGenerator
var once sync.Once

func GetInstance() *IdGenerator {
	once.Do(func() {
		instance = &IdGenerator{}
	})
	return instance
}
