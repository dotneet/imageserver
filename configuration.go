// Package advanced provides an advanced example.
package imageserver

type Configuration struct {
	Bind    string
	Parsers []string
	Cache   CacheConfiguration
	Source  SourceConfiguration
}

type CacheConfiguration struct {
	Redis      RedisConfiguration
	File       FileCacheConfiguration
	Memory     MemoryCacheConfiguration
	Memcached  MemcachedConfiguration
	Groupcache GroupcacheConfiguration
}

type RedisConfiguration struct {
	Host string
}

type MemcachedConfiguration struct {
	Host string
}

type GroupcacheConfiguration struct {
	Peers     string
	Name      string
	MaxSize   int64
	StatsPath string
}

type MemoryCacheConfiguration struct {
	MaxSize int64
}

type FileCacheConfiguration struct {
	Path string
}

type SourceConfiguration struct {
	Http HttpSourceConfiguration
	File FileSourceConfiguration
}

type HttpSourceConfiguration struct {
	UrlPrefix string
}

type FileSourceConfiguration struct {
	Path string
}
