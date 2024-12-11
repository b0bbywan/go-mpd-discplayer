package mounts

import (
	"fmt"
	"sync"
)

type protectedCache struct {
	cache map[string]string
	mu    sync.RWMutex // Protects access to cache
}

func newCache() *protectedCache {
	return &protectedCache{
		cache: make(map[string]string),
	}
}

func (c *protectedCache) AddCache(source, target string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[source] = target
}

func (c *protectedCache) RemoveCache(source string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, source)
}

func (c *protectedCache) GetCache(source string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if val, exists := c.cache[source]; exists {
		return val, nil
	}
	return "", fmt.Errorf("%s mount point does not exist in cache", source)
}
