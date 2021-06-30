package emlogrus

import (
	"github.com/go-emix/utils"
	frl "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"strings"
	"time"
)

type LogOutType string

type LogLevel string

func (r LogLevel) Logrus() logrus.Level {
	switch r {
	case DebugLog:
		return logrus.DebugLevel
	case InfoLog:
		return logrus.InfoLevel
	case WarnLog:
		return logrus.WarnLevel
	case ErrorLog:
		return logrus.ErrorLevel
	default:
		panic("unknown level")
	}
}

type LogFormat string

func (r LogFormat) Logrus() logrus.Formatter {
	switch r {
	case TextLog:
		return &logrus.TextFormatter{}
	case JsonLog:
		return &logrus.JSONFormatter{}
	default:
		panic("unknown format")
	}
}

const (
	ConsoleOut LogOutType = "console"
	FileOut               = "file"
	DebugLog   LogLevel   = "debug"
	InfoLog               = "info"
	WarnLog               = "warn"
	ErrorLog              = "error"
	TextLog    LogFormat  = "text"
	JsonLog               = "json"
)

type EmixConfig struct {
	Log []LogConfig `yaml:"log,flow"`
}

type RootConfig struct {
	Emix EmixConfig `yaml:"emix"`
}

type LogConfig struct {
	Level       LogLevel   `yaml:"level"`
	Format      LogFormat  `yaml:"format"`
	OutType     LogOutType `yaml:"outType"`
	OutDir      string     `yaml:"outDir"`
	MaxAge      int        `yaml:"maxAge"`
	MaxCount    uint       `yaml:"maxCount"`
	SingleLevel bool       `yaml:"singleLevel"`
	Disabled    bool       `yaml:"disabled"`
}

func (r LogConfig) Option() Option {
	return Option{
		Level:       r.Level.Logrus(),
		Format:      r.Format.Logrus(),
		OutType:     r.OutType,
		OutDir:      r.OutDir,
		MaxAge:      time.Duration(r.MaxAge) * time.Hour,
		MaxCount:    r.MaxCount,
		SingleLevel: r.SingleLevel,
		Disabled:    r.Disabled,
	}
}

type Operation struct {
	Lcs []LogConfig
}

func (r Operation) Setup() {
	if len(r.Lcs) != 0 {
		_log = NewLogEntry(r.Lcs...)
	}
}

var GlobalLevel = DebugLog

func AfterInit(yamlFile string, cs ...LogConfig) (oper Operation) {
	if len(cs) != 0 {
		oper.Lcs = cs
		return
	}
	if yamlFile != "" && utils.FileIsExist(yamlFile) {
		lcs := parseYaml(yamlFile)
		oper.Lcs = lcs
	}
	return
}

func parseYaml(f string) []LogConfig {
	bytes, e := os.ReadFile(f)
	utils.PanicError(e)
	yc := RootConfig{}
	e = yaml.Unmarshal(bytes, &yc)
	utils.PanicError(e)
	return yc.Emix.Log
}

type Option struct {
	Level       logrus.Level
	Format      logrus.Formatter
	OutType     LogOutType
	OutDir      string
	MaxAge      time.Duration
	MaxCount    uint
	SingleLevel bool
	Disabled    bool
}

type LogEntry struct {
	logs []*OptionLogger
}

func NewLogEntry(lcs ...LogConfig) *LogEntry {
	logs := make([]*OptionLogger, 0)
	for _, o := range lcs {
		logger := newOptionLogger(o.Option())
		if logger != nil {
			logs = append(logs, logger)
		}
	}
	return &LogEntry{logs: logs}
}

func NewLogEntryFromOption(ops ...Option) *LogEntry {
	logs := make([]*OptionLogger, 0)
	for _, o := range ops {
		logger := newOptionLogger(o)
		if logger != nil {
			logs = append(logs, logger)
		}
	}
	return &LogEntry{logs: logs}
}

func (l *LogEntry) Debugf(format string, args ...interface{}) {
	if logrus.DebugLevel > GlobalLevel.Logrus() {
		return
	}
	l.batchLogf(logrus.DebugLevel, format, args...)
}

func (l *LogEntry) Infof(format string, args ...interface{}) {
	if logrus.InfoLevel > GlobalLevel.Logrus() {
		return
	}
	l.batchLogf(logrus.InfoLevel, format, args...)
}

func (l *LogEntry) Warnf(format string, args ...interface{}) {
	if logrus.WarnLevel > GlobalLevel.Logrus() {
		return
	}
	l.batchLogf(logrus.WarnLevel, format, args...)
}

func (l *LogEntry) Errorf(format string, args ...interface{}) {
	if logrus.ErrorLevel > GlobalLevel.Logrus() {
		return
	}
	l.batchLogf(logrus.ErrorLevel, format, args...)
}

type Fields map[string]interface{}

func (l *LogEntry) DebugfWith(fields map[string]interface{}, format string, args ...interface{}) {
	if logrus.DebugLevel > GlobalLevel.Logrus() {
		return
	}
	for _, v := range l.withs(logrus.DebugLevel, fields) {
		v.Debugf(format, args...)
	}
}

func (l *LogEntry) InfofWith(fields map[string]interface{}, format string, args ...interface{}) {
	if logrus.InfoLevel > GlobalLevel.Logrus() {
		return
	}
	for _, v := range l.withs(logrus.InfoLevel, fields) {
		v.Infof(format, args...)
	}
}

func (l *LogEntry) WarnfWith(fields map[string]interface{}, format string, args ...interface{}) {
	if logrus.WarnLevel > GlobalLevel.Logrus() {
		return
	}
	for _, v := range l.withs(logrus.WarnLevel, fields) {
		v.Warnf(format, args...)
	}
}

func (l *LogEntry) ErrorfWith(fields map[string]interface{}, format string, args ...interface{}) {
	if logrus.ErrorLevel > GlobalLevel.Logrus() {
		return
	}
	for _, v := range l.withs(logrus.ErrorLevel, fields) {
		v.Errorf(format, args...)
	}
}

func (l *LogEntry) DebugWith(fields map[string]interface{}, args ...interface{}) {
	if logrus.DebugLevel > GlobalLevel.Logrus() {
		return
	}
	for _, v := range l.withs(logrus.DebugLevel, fields) {
		v.Debugln(args...)
	}
}

func (l *LogEntry) InfoWith(fields map[string]interface{}, args ...interface{}) {
	if logrus.InfoLevel > GlobalLevel.Logrus() {
		return
	}
	for _, v := range l.withs(logrus.InfoLevel, fields) {
		v.Infoln(args...)
	}
}

func (l *LogEntry) WarnWith(fields map[string]interface{}, args ...interface{}) {
	if logrus.WarnLevel > GlobalLevel.Logrus() {
		return
	}
	for _, v := range l.withs(logrus.WarnLevel, fields) {
		v.Warnln(args...)
	}
}

func (l *LogEntry) ErrorWith(fields map[string]interface{}, args ...interface{}) {
	if logrus.ErrorLevel > GlobalLevel.Logrus() {
		return
	}
	for _, v := range l.withs(logrus.ErrorLevel, fields) {
		v.Errorln(args...)
	}
}

func (l *LogEntry) withs(level logrus.Level, fields map[string]interface{}) []*logrus.Entry {
	entries := make([]*logrus.Entry, 0)
	for _, v := range l.logs {
		if v.Op.SingleLevel && v.Op.Level != level {
			continue
		}
		e := v.WithFields(fields)
		entries = append(entries, e)
	}
	return entries
}

func (l *LogEntry) Debug(args ...interface{}) {
	if logrus.DebugLevel > GlobalLevel.Logrus() {
		return
	}
	l.batchLogln(logrus.DebugLevel, args...)
}

func (l *LogEntry) Info(args ...interface{}) {
	if logrus.InfoLevel > GlobalLevel.Logrus() {
		return
	}
	l.batchLogln(logrus.InfoLevel, args...)
}

func (l *LogEntry) Warn(args ...interface{}) {
	if logrus.WarnLevel > GlobalLevel.Logrus() {
		return
	}
	l.batchLogln(logrus.WarnLevel, args...)
}

func (l *LogEntry) Error(args ...interface{}) {
	if logrus.ErrorLevel > GlobalLevel.Logrus() {
		return
	}
	l.batchLogln(logrus.ErrorLevel, args...)
}

func (l *LogEntry) batchLogln(level logrus.Level, args ...interface{}) {
	for _, v := range l.logs {
		v.Logln(level, args...)
	}
}

func (l *LogEntry) batchLogf(level logrus.Level, format string, args ...interface{}) {
	for _, v := range l.logs {
		v.Logf(level, format, args...)
	}
}

var _log *LogEntry

func Debugf(format string, args ...interface{}) {
	_log.Debugf(format, args...)
}
func Infof(format string, args ...interface{}) {
	_log.Infof(format, args...)
}

func Warnf(format string, args ...interface{}) {
	_log.Warnf(format, args...)
}
func Errorf(format string, args ...interface{}) {
	_log.Errorf(format, args...)
}

func Debug(args ...interface{}) {
	_log.Debug(args...)
}

func Info(args ...interface{}) {
	_log.Info(args...)
}

func Warn(args ...interface{}) {
	_log.Warn(args...)
}

func Error(args ...interface{}) {
	_log.Error(args...)
}

func DebugWith(fields map[string]interface{}, args ...interface{}) {
	_log.DebugWith(fields, args...)
}

func InfoWith(fields map[string]interface{}, args ...interface{}) {
	_log.InfoWith(fields, args...)
}

func WarnWith(fields map[string]interface{}, args ...interface{}) {
	_log.WarnWith(fields, args...)
}

func ErrorWith(fields map[string]interface{}, args ...interface{}) {
	_log.ErrorWith(fields, args...)
}

func DebugfWith(fields map[string]interface{}, format string, args ...interface{}) {
	_log.DebugfWith(fields, format, args...)
}

func InfofWith(fields map[string]interface{}, format string, args ...interface{}) {
	_log.InfofWith(fields, format, args...)
}

func WarnfWith(fields map[string]interface{}, format string, args ...interface{}) {
	_log.WarnfWith(fields, format, args...)
}

func ErrorfWith(fields map[string]interface{}, format string, args ...interface{}) {
	_log.ErrorfWith(fields, format, args...)
}

func init() {
	var ymlFile = "config.yml"
	var yamlFile = "config.yaml"
	var lcs = make([]LogConfig, 0)
	if !utils.FileIsExist(ymlFile) {
		if utils.FileIsExist(yamlFile) {
			lcs = parseYaml(yamlFile)
		}
	} else {
		lcs = parseYaml(ymlFile)
	}
	if len(lcs) == 0 {
		option := Option{
			Level:   logrus.DebugLevel,
			Format:  &logrus.TextFormatter{},
			OutType: ConsoleOut,
		}
		_log = NewLogEntryFromOption(option)
		return
	}
	_log = NewLogEntry(lcs...)
}

type OptionLogger struct {
	*logrus.Logger
	Op Option
}

func (r *OptionLogger) Log(level logrus.Level, args ...interface{}) {
	if r.Op.SingleLevel {
		if r.Op.Level == level {
			r.Logger.Log(level, args...)
		}
	} else {
		r.Logger.Log(level, args...)
	}
}

func (r *OptionLogger) Logf(level logrus.Level, format string, args ...interface{}) {
	if r.Op.SingleLevel {
		if r.Op.Level == level {
			r.Logger.Logf(level, format, args...)
		}
	} else {
		r.Logger.Logf(level, format, args...)
	}
}

func (r *OptionLogger) Logln(level logrus.Level, args ...interface{}) {
	if r.Op.SingleLevel {
		if r.Op.Level == level {
			r.Logger.Logln(level, args...)
		}
	} else {
		r.Logger.Logln(level, args...)
	}
}

func newOptionLogger(op Option) *OptionLogger {
	if op.Disabled {
		return nil
	}
	writer := outTypeToWriter(op)
	if writer == nil {
		return nil
	}
	log := logrus.New()
	log.SetFormatter(op.Format)
	log.SetOutput(writer)
	log.SetLevel(op.Level)
	return &OptionLogger{
		Logger: log,
		Op:     op,
	}
}

func outTypeToWriter(c Option) io.Writer {
	switch c.OutType {
	case ConsoleOut:
		return os.Stdout
	case FileOut:
		file := c.OutDir
		if file == "" {
			file = "log/" + c.Level.String()
		}
		le := len(file)
		if ind := strings.LastIndex(file, "/"); ind == le-1 {
			file = file[:le-2]
		}
		if utils.FileIsExist(file) {
			err := os.MkdirAll(file, os.ModePerm)
			utils.PanicError(err)
		}
		if c.MaxAge != 0 {
			c.MaxCount = 0
		}
		logf, err := frl.New(
			file+"/"+"%Y-%m-%d.log",
			frl.WithRotationTime(24*time.Hour),
			frl.WithMaxAge(c.MaxAge),
			frl.WithRotationCount(c.MaxCount),
		)
		utils.PanicError(err)
		return logf
	}
	return nil
}
