// Package advanced provides an advanced example.
package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"
	"time"

	"encoding/json"
	"github.com/bradfitz/gomemcache/memcache"
	"github.com/disintegration/gift"
	"github.com/garyburd/redigo/redis"
	"github.com/golang/groupcache"
	imageserver "github.com/pierrre/imageserver"
	imageserver_cache "github.com/pierrre/imageserver/cache"
	imageserver_cache_file "github.com/pierrre/imageserver/cache/file"
	imageserver_cache_groupcache "github.com/pierrre/imageserver/cache/groupcache"
	imageserver_cache_memcache "github.com/pierrre/imageserver/cache/memcache"
	imageserver_cache_memory "github.com/pierrre/imageserver/cache/memory"
	imageserver_cache_redis "github.com/pierrre/imageserver/cache/redis"
	imageserver_http "github.com/pierrre/imageserver/http"
	imageserver_http_crop "github.com/pierrre/imageserver/http/crop"
	imageserver_http_gamma "github.com/pierrre/imageserver/http/gamma"
	imageserver_http_gift "github.com/pierrre/imageserver/http/gift"
	imageserver_http_image "github.com/pierrre/imageserver/http/image"
	imageserver_image "github.com/pierrre/imageserver/image"
	_ "github.com/pierrre/imageserver/image/bmp"
	imageserver_image_crop "github.com/pierrre/imageserver/image/crop"
	imageserver_image_gamma "github.com/pierrre/imageserver/image/gamma"
	imageserver_image_gif "github.com/pierrre/imageserver/image/gif"
	imageserver_image_gift "github.com/pierrre/imageserver/image/gift"
	_ "github.com/pierrre/imageserver/image/jpeg"
	_ "github.com/pierrre/imageserver/image/png"
	_ "github.com/pierrre/imageserver/image/tiff"
	imageserver_source_file "github.com/pierrre/imageserver/source/file"
	imageserver_source_http "github.com/pierrre/imageserver/source/http"
	"gopkg.in/yaml.v2"
	"net/url"
	"strings"
)

var (
	config     imageserver.Configuration
	flagConfig = "config.yml"
	flagCache  = int64(128 * (1 << 20))
)

func main() {
	if err := parseFlags(); err != nil {
		panic(err)
	}
	startHTTPServer()
}

func parseFlags() error {
	flag.StringVar(&flagConfig, "config", flagConfig, "ConfigFile")
	flag.StringVar(&flagConfig, "c", flagConfig, "ConfigFile")

	flag.Parse()

	data, err := ioutil.ReadFile(flagConfig)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}

	return nil
}

func startHTTPServer() {
	fmt.Printf("Start Server.\nlistening on %s\n", config.Bind)
	err := http.ListenAndServe(config.Bind, newHTTPHandler())

	if err != nil {
		panic(err)
	}
}

func newHTTPHandler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", http.StripPrefix("/", newImageHTTPHandler()))
	mux.Handle("/favicon.ico", http.NotFoundHandler())
	if config.Cache.Groupcache.StatsPath != "" {
		mux.HandleFunc("/stats", groupcacheStatsHTTPHandler)
	}
	return mux
}

func newImageHTTPHandler() http.Handler {

	parsers := imageserver_http.ListParser([]imageserver_http.Parser{
		&imageserver_http.SourcePathParser{},
		&imageserver_http_crop.Parser{},
		&imageserver_http_gift.RotateParser{},
		&imageserver_http_gift.ResizeParser{},
		&imageserver_http_image.FormatParser{},
		&imageserver_http_image.QualityParser{},
		&imageserver_http_gamma.CorrectionParser{},
	})

	if config.Source.Http.UrlPrefix != "" {
		parsers = append(parsers, &imageserver_http.SourcePrefixParser{
			Parser: &imageserver_http.SourcePathParser{},
			Prefix: config.Source.Http.UrlPrefix,
		})
	}

	var handler http.Handler
	handler = &imageserver_http.Handler{
		Parser:   parsers,
		Server:   newServer(),
		ETagFunc: imageserver_http.NewParamsHashETagFunc(sha256.New),
	}
	handler = &imageserver_http.ExpiresHandler{
		Handler: handler,
		Expires: 7 * 24 * time.Hour,
	}
	handler = &imageserver_http.CacheControlPublicHandler{
		Handler: handler,
	}
	return handler
}

func newServer() imageserver.Server {

	var srv imageserver.Server = &imageserver_source_http.Server{}
	srv = newServerFile(srv)
	srv = newServerImage(srv)
	srv = newServerLimit(srv)
	srv = newServerFileCache(srv)
	srv = newServerGroupcache(srv)
	srv = newServerRedis(srv)
	srv = newServerMemcache(srv)
	srv = newServerCacheMemory(srv)

	return srv
}

func newServerFile(srv imageserver.Server) imageserver.Server {
	if config.Source.File.Path == "" {
		return srv
	}
	return &imageserver_source_file.Server{Root: config.Source.File.Path}
}

func newServerImage(srv imageserver.Server) imageserver.Server {
	basicHdr := &imageserver_image.Handler{
		Processor: imageserver_image_gamma.NewCorrectionProcessor(
			imageserver_image.ListProcessor([]imageserver_image.Processor{
				&imageserver_image_crop.Processor{},
				&imageserver_image_gift.RotateProcessor{
					DefaultInterpolation: gift.CubicInterpolation,
				},
				&imageserver_image_gift.ResizeProcessor{
					DefaultResampling: gift.LanczosResampling,
					MaxWidth:          2048,
					MaxHeight:         2048,
				},
			}),
			true,
		),
	}
	gifHdr := &imageserver_image_gif.FallbackHandler{
		Handler: &imageserver_image_gif.Handler{
			Processor: &imageserver_image_gif.SimpleProcessor{
				Processor: imageserver_image.ListProcessor([]imageserver_image.Processor{
					&imageserver_image_crop.Processor{},
					&imageserver_image_gift.RotateProcessor{
						DefaultInterpolation: gift.NearestNeighborInterpolation,
					},
					&imageserver_image_gift.ResizeProcessor{
						DefaultResampling: gift.NearestNeighborResampling,
						MaxWidth:          1024,
						MaxHeight:         1024,
					},
				}),
			},
		},
		Fallback: basicHdr,
	}
	return &imageserver.HandlerServer{
		Server:  srv,
		Handler: gifHdr,
	}
}

func newServerLimit(srv imageserver.Server) imageserver.Server {
	return imageserver.NewLimitServer(srv, runtime.GOMAXPROCS(0)*2)
}

func newServerCacheMemory(srv imageserver.Server) imageserver.Server {
	if config.Cache.Memory.MaxSize <= 0 {
		return srv
	}
	return &imageserver_cache.Server{
		Server:       srv,
		Cache:        imageserver_cache_memory.New(config.Cache.Memory.MaxSize),
		KeyGenerator: imageserver_cache.NewParamsHashKeyGenerator(sha256.New),
	}
}

func newServerMemcache(srv imageserver.Server) imageserver.Server {
	if config.Cache.Memcached.Host == "" {
		return srv
	}
	cl := memcache.New(config.Cache.Memcached.Host)
	var cch imageserver_cache.Cache
	cch = &imageserver_cache_memcache.Cache{Client: cl}
	cch = &imageserver_cache.IgnoreError{Cache: cch}
	cch = &imageserver_cache.Async{Cache: cch}
	kg := imageserver_cache.NewParamsHashKeyGenerator(sha256.New)
	return &imageserver_cache.Server{
		Server:       srv,
		Cache:        cch,
		KeyGenerator: kg,
	}
}

func newServerRedis(srv imageserver.Server) imageserver.Server {
	if config.Cache.Redis.Host == "" {
		return srv
	}
	pool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", config.Cache.Redis.Host)
		},
		MaxIdle: 50,
	}
	var cch imageserver_cache.Cache
	cch = &imageserver_cache_redis.Cache{
		Pool:   pool,
		Expire: 7 * 24 * time.Hour,
	}
	cch = &imageserver_cache.IgnoreError{Cache: cch}
	cch = &imageserver_cache.Async{Cache: cch}
	var kg imageserver_cache.KeyGenerator
	kg = imageserver_cache.NewParamsHashKeyGenerator(sha256.New)
	kg = &imageserver_cache.PrefixKeyGenerator{
		KeyGenerator: kg,
		Prefix:       "image:",
	}
	return &imageserver_cache.Server{
		Server:       srv,
		Cache:        cch,
		KeyGenerator: kg,
	}
}

func newServerFileCache(srv imageserver.Server) imageserver.Server {
	if config.Cache.File.Path == "" {
		return srv
	}
	cch := imageserver_cache_file.Cache{Path: config.Cache.File.Path}
	kg := imageserver_cache.NewParamsHashKeyGenerator(sha256.New)
	return &imageserver_cache.Server{
		Server:       srv,
		Cache:        &cch,
		KeyGenerator: kg,
	}
}

func newServerGroupcache(srv imageserver.Server) imageserver.Server {
	if config.Cache.Groupcache.Peers == "" {
		return srv
	}
	return imageserver_cache_groupcache.NewServer(
		srv,
		imageserver_cache.NewParamsHashKeyGenerator(sha256.New),
		config.Cache.Groupcache.Name,
		config.Cache.Groupcache.MaxSize,
	)
}

func initGroupcacheHTTPPool() {
	self := (&url.URL{Scheme: "http", Host: config.Bind}).String()
	var peers []string
	peers = append(peers, self)
	for _, p := range strings.Split(config.Cache.Groupcache.Peers, ",") {
		if p == "" {
			continue
		}
		peer := (&url.URL{Scheme: "http", Host: p}).String()
		peers = append(peers, peer)
	}
	pool := groupcache.NewHTTPPool(self)
	pool.Context = imageserver_cache_groupcache.HTTPPoolContext
	pool.Transport = imageserver_cache_groupcache.NewHTTPPoolTransport(nil)
	pool.Set(peers...)
}

func groupcacheStatsHTTPHandler(w http.ResponseWriter, req *http.Request) {
	gp := groupcache.GetGroup(config.Cache.Groupcache.Name)
	if gp == nil {
		http.Error(w, fmt.Sprintf("group %s not found", config.Cache.Groupcache.Name), http.StatusServiceUnavailable)
		return
	}
	type cachesStats struct {
		Main groupcache.CacheStats
		Hot  groupcache.CacheStats
	}
	type stats struct {
		Group  groupcache.Stats
		Caches cachesStats
	}
	data, err := json.MarshalIndent(
		stats{
			Group: gp.Stats,
			Caches: cachesStats{
				Main: gp.CacheStats(groupcache.MainCache),
				Hot:  gp.CacheStats(groupcache.HotCache),
			},
		},
		"",
		"	",
	)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(data)
}
