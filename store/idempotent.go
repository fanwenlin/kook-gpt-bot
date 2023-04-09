package store

import (
	"context"
	"sync"

	"github.com/coocood/freecache"
)

var cache *freecache.Cache
var tmux = sync.Mutex{}

func init() {
	cacheSize := 100 * 1024 * 1024
	cache = freecache.NewCache(cacheSize)
}

func CheckIdempotent(ctx context.Context, msgID string) bool {
	tmux.Lock()
	defer tmux.Unlock()

	var err error

	_, err = cache.Get([]byte(msgID))
	if err == nil {
		return false
	}
	cache.Set([]byte(msgID), []byte("1"), 111)
	return true
}
