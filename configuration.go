// Package advanced provides an advanced example.
package imageserver

type Configuration struct {
	Bind       string
	Processor  ProcessorConfiguration
	Cache      CacheConfiguration
	Source     SourceConfiguration
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
	MaxSize   int64		`yaml:"maxSize"`
	StatsPath string	`yaml:"statsPath"`
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
	UrlPrefix string	`yaml:"urlPrefix"`
}

type FileSourceConfiguration struct {
	Path string
}

type ProcessorConfiguration struct {
	Resize ResizeProcessorConfiguration
	Pngquant PngquantProcessorConfiguration
}

type ResizeProcessorConfiguration struct {
	MaxWidth      int	`yaml:"maxWidth"`
	MaxHeight     int	`yaml:"maxHeight"`
	EnableMaxArea int	`yaml:"enableMaxArea"`
}

type PngquantProcessorConfiguration struct {
	Command 	string
	Speed		string
	EnableMaxArea int	`yaml:"enableMaxArea"`
}
