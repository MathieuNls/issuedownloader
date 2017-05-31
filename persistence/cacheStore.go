package persistence

import (
	"strconv"
	"sync"
)

type Cache interface {
	Put(store string, key interface{}, value interface{})
	Fetch(store string, key interface{}) interface{}
	FetchWithCB(store string, key interface{}, callback func(interface{}) interface{}) interface{}
	Stats(store string) stat
}

type stat struct {
	Name     string
	Elements int
	Hits     int
	Misses   int
}

var instance Cache
var once sync.Once
var mutex *sync.Mutex

func GetCacheInstance() Cache {
	once.Do(func() {
		instance = &memoryCache{
			stores: make(map[string]map[interface{}]interface{}),
			stats:  make(map[string]*stat),
		}
		mutex = &sync.Mutex{}
	})
	return instance
}

func (stat stat) String() string {
	return stat.Name + "\n" +
		" - Elements:" + strconv.Itoa(stat.Elements) + "\n" +
		" - Hits:" + strconv.Itoa(stat.Hits) + "\n" +
		" - Misses:" + strconv.Itoa(stat.Misses)
}

type memoryCache struct {
	stores map[string]map[interface{}]interface{}
	stats  map[string]*stat
}

func (mc *memoryCache) Stats(store string) stat {
	return *mc.stats[store]
}

func (mc *memoryCache) Put(store string, key interface{}, value interface{}) {

	mutex.Lock()
	if memoryStore, present := mc.stores[store]; present {

		memoryStore[key] = value
	} else {

		mc.createStore(store)
		mc.stores[store][key] = value
	}

	mc.stats[store].Elements++
	mutex.Unlock()
}

func (mc *memoryCache) createStore(store string) {
	mc.stores[store] = make(map[interface{}]interface{})
	mc.stats[store] = &stat{
		Name:     store,
		Elements: 0,
		Hits:     0,
		Misses:   0,
	}
}

func (mc *memoryCache) FetchWithCB(store string, key interface{}, callback func(interface{}) interface{}) interface{} {

	tmp := mc.Fetch(store, key)

	if tmp != nil {
		return callback(tmp)
	}
	return tmp
}

func (mc *memoryCache) Fetch(store string, key interface{}) interface{} {

	mutex.Lock()
	var value interface{}
	if memoryStore, present := mc.stores[store]; present {
		value = memoryStore[key]
	} else {

		mc.createStore(store)
		mc.stats[store].Misses++
	}
	mutex.Unlock()

	return value
}
