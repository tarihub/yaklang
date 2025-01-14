package ssaapi

import (
	"io"
	"strings"
	"time"

	"github.com/yaklang/yaklang/common/utils"
	"github.com/yaklang/yaklang/common/utils/filesys"
	"github.com/yaklang/yaklang/common/yak/ssa"
)

type config struct {
	language        Language
	Builder         Builder
	feedCode        bool
	ignoreSyntaxErr bool

	// input, code or project path
	code io.Reader
	// project
	fs        filesys.FileSystem
	entryFile []string
	// entryPath []string

	externLib    map[string]map[string]any
	externValue  map[string]any
	defineFunc   map[string]any
	externMethod ssa.MethodBuilder

	DatabaseProgramName        string
	DatabaseProgramCacheHitter func(any)
	// for hash
	externInfo string
}

func defaultConfig() *config {
	return &config{
		language:                   "",
		Builder:                    nil,
		code:                       nil,
		fs:                         filesys.NewLocalFs(),
		entryFile:                  make([]string, 0),
		externLib:                  make(map[string]map[string]any),
		externValue:                make(map[string]any),
		defineFunc:                 make(map[string]any),
		DatabaseProgramCacheHitter: func(any) {},
	}
}

func (c *config) CaclHash() string {
	return utils.CalcSha1(c.code, c.language, c.ignoreSyntaxErr, c.externInfo)
}

type Option func(*config)

func WithLanguage(language Language) Option {
	return func(c *config) {
		c.language = language
		if parser, ok := LanguageBuilders[language]; ok {
			c.Builder = parser
		} else {
			c.Builder = nil
		}
	}
}

func WithFileSystemEntry(files ...string) Option {
	return func(c *config) {
		c.entryFile = append(c.entryFile, files...)
	}
}

func WithExternLib(name string, table map[string]any) Option {
	return func(c *config) {
		c.externLib[name] = table
	}
}

func WithExternValue(table map[string]any) Option {
	return func(c *config) {
		// c.externValue = table
		for name, value := range table {
			// this value set again
			// if _, ok := c.externValue[name]; !ok {
			// 	// skip
			// }
			c.externValue[name] = value
		}
	}
}

func WithExternMethod(b ssa.MethodBuilder) Option {
	return func(c *config) {
		c.externMethod = b
	}
}

func WithIgnoreSyntaxError(b ...bool) Option {
	return func(c *config) {
		if len(b) > 1 {
			c.ignoreSyntaxErr = b[0]
		} else {
			c.ignoreSyntaxErr = true
		}
	}
}

func WithExternInfo(info string) Option {
	return func(c *config) {
		c.externInfo = info
	}
}

func WithDefineFunc(table map[string]any) Option {
	return func(c *config) {
		for name, t := range table {
			c.defineFunc[name] = t
		}
	}
}

func WithFeedCode(b ...bool) Option {
	return func(c *config) {
		if len(b) > 1 {
			c.feedCode = b[0]
		} else {
			c.feedCode = true
		}
	}
}

// save to database, please set the program name
func WithDatabaseProgramName(name string) Option {
	return func(c *config) {
		c.DatabaseProgramName = name
	}
}

func WithDatabaseProgramCacheHitter(h func(i any)) Option {
	return func(c *config) {
		c.DatabaseProgramCacheHitter = h
	}
}

func ParseProjectFromPath(path string, opts ...Option) ([]*Program, error) {
	fs := filesys.NewLocalFsWithPath(path)
	return ParseProject(fs, opts...)
}

func ParseProject(fs filesys.FileSystem, opts ...Option) ([]*Program, error) {
	config := defaultConfig()
	for _, opt := range opts {
		opt(config)
	}
	config.fs = fs
	if config.fs == nil {
		return nil, utils.Errorf("need set filesystem")
	}
	ret, err := config.parseProject()
	return ret, err
}

var ttlSSAParseCache = utils.NewTTLCache[*Program](30 * time.Minute)

func ClearCache() {
	ttlSSAParseCache.Purge()
}

// Parse parse code to ssa.Program
func Parse(code string, opts ...Option) (*Program, error) {
	input := strings.NewReader(code)
	return ParseFromReader(input, opts...)
}

// ParseFromReader parse simple file to ssa.Program
func ParseFromReader(input io.Reader, opts ...Option) (*Program, error) {
	config := defaultConfig()
	for _, opt := range opts {
		opt(config)
	}
	config.code = input

	hash := config.CaclHash()
	if prog, ok := ttlSSAParseCache.Get(hash); ok {
		return prog, nil
	} else {
		ret, err := config.parseFile()
		if err != nil {
			ttlSSAParseCache.SetWithTTL(hash, ret, 30*time.Minute)
		}
		return ret, err
	}
}

func (p *Program) Feed(code io.Reader) error {
	if p.config == nil || !p.config.feedCode || p.config.Builder == nil {
		return utils.Errorf("not support language %s", p.config.language)
	}
	return p.config.feed(p.Program, code)
}

func FromDatabase(programName string, opts ...Option) (*Program, error) {
	config := defaultConfig()
	for _, opt := range opts {
		opt(config)
	}
	config.DatabaseProgramName = programName
	return config.fromDatabase()
}

var Exports = map[string]any{
	"Parse": Parse,

	"withLanguage":            WithLanguage,
	"withExternLib":           WithExternLib,
	"withExternValue":         WithExternValue,
	"withDatabaseProgramName": WithDatabaseProgramName,
	// language:
	"Javascript": JS,
	"Yak":        Yak,
	"PHP":        PHP,
	"Java":       JAVA,
}
