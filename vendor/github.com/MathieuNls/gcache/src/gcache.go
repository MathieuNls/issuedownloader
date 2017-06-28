package gcache

import (
	"strconv"
	"sync"
)

//Cache defines what caching do
type Cache interface {
	Put(store string, key interface{}, value interface{})
	Fetch(store string, key interface{}) interface{}
	FetchWithCB(store string, key interface{}, callback func(interface{}) interface{}) interface{}
	Stats(store string) stat
}

//stat contains stats for our cache stores
type stat struct {
	Name     string
	Elements int
	Hits     int
	Misses   int
}

var instance Cache
var once sync.Once
var mutex *sync.RWMutex

//GetCacheInstance returns a Singleton of a Cache store
func GetCacheInstance() Cache {
	once.Do(func() {
		instance = &memoryCache{
			stores: make(map[string]map[interface{}]interface{}),
			stats:  make(map[string]*stat),
		}
		mutex = &sync.RWMutex{}
	})
	return instance
}

//String returns string representation of a stat struct
func (stat stat) String() string {
	return stat.Name + "\n" +
		" - Elements:" + strconv.Itoa(stat.Elements) + "\n" +
		" - Hits:" + strconv.Itoa(stat.Hits) + "\n" +
		" - Misses:" + strconv.Itoa(stat.Misses)
}

//memoryCache is an implementation of Cache
type memoryCache struct {
	stores map[string]map[interface{}]interface{}
	stats  map[string]*stat
}

//Stats returns the stats of a given cachestore
func (mc *memoryCache) Stats(store string) stat {
	return *mc.stats[store]
}

//Put puts a new key/value into the given store.
//If the store doesn t exist, it'll be created
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

//createStore creates a new store and associated stats struct
func (mc *memoryCache) createStore(store string) {
	mc.stores[store] = make(map[interface{}]interface{})
	mc.stats[store] = &stat{
		Name:     store,
		Elements: 0,
		Hits:     0,
		Misses:   0,
	}
}

//FetchWithCB fetches an element with a callback
func (mc *memoryCache) FetchWithCB(store string, key interface{}, callback func(interface{}) interface{}) interface{} {

	tmp := mc.Fetch(store, key)

	if tmp != nil && callback != nil {
		return callback(tmp)
	}
	return tmp
}

//Fetch fetches an element
func (mc *memoryCache) Fetch(store string, key interface{}) interface{} {

	mutex.Lock()
	var value interface{}
	if memoryStore, present := mc.stores[store]; present {
		value = memoryStore[key]
		mc.stats[store].Hits++
	} else {

		mc.createStore(store)
		mc.stats[store].Misses++
	}
	mutex.Unlock()

	return value
}
