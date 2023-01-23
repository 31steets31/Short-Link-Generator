package cache_manager

import (
	"errors"
	"sync"
	"time"
)

type Cache struct {
	sync.RWMutex
	defaultExpiration time.Duration
	cleanupTime       time.Duration
	data              map[string]Value
}

type Value struct {
	CreateTime time.Time
	Expiration int64
	Value      string
}

func CacheCreate(defaultExpiration, cleanupTime time.Duration) *Cache {

	data := make(map[string]Value)

	cache := Cache{
		data:              data,
		defaultExpiration: defaultExpiration,
		cleanupTime:       cleanupTime,
	}

	if cleanupTime > 0 {
		cache.startGC()
	}

	return &cache
}

func (c *Cache) Set(key string, value string, duration time.Duration) {

	var expiration int64

	if duration == 0 {
		duration = c.defaultExpiration
	}

	if duration > 0 {
		expiration = time.Now().Add(duration).UnixNano()
	}

	c.Lock()
	defer c.Unlock()

	c.data[key] = Value{
		Value:      value,
		Expiration: expiration,
		CreateTime: time.Now(),
	}

}

func (c *Cache) Get(key string) (string, bool) {

	c.RLock()
	defer c.RUnlock()

	item, found := c.data[key]

	if !found {
		return "", false
	}

	if item.Expiration > 0 &&
		time.Now().UnixNano() > item.Expiration {
		return "", false
	}

	return item.Value, true
}

func (c *Cache) Delete(key string) error {

	c.Lock()
	defer c.Unlock()

	if _, found := c.data[key]; !found {
		return errors.New("error: Key not found")
	}

	delete(c.data, key)

	return nil
}

func (c *Cache) startGC() {
	go c.gC()
}

func (c *Cache) gC() {

	for {
		<-time.After(c.cleanupTime)

		if c.data == nil {
			return
		}

		if keys := c.expiredKeys(); len(keys) != 0 {
			c.clearValues(keys)
		}
	}
}

func (c *Cache) expiredKeys() (keys []string) {

	c.RLock()
	defer c.RUnlock()

	for k, i := range c.data {
		if i.Expiration > 0 &&
			time.Now().UnixNano() > i.Expiration {
			keys = append(keys, k)
		}
	}

	return
}

func (c *Cache) clearValues(keys []string) {

	c.Lock()
	defer c.Unlock()

	for _, k := range keys {
		delete(c.data, k)
	}
}
