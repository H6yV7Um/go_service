package utils

import (
	"strings"
	"fmt"
	log "github.com/cihub/seelog"
	"path/filepath"
)

// DefaultLogger 全局的logger实例，在Service方法之外使用
var DefaultLogger log.LoggerInterface

// MonitorLogger
var MonitorLogger log.LoggerInterface

// ErrorLogNumber 用于错误日志计数
var ErrorLogNumber = 0
var ErrorLogNumberIncreasement = 0

func InitLogger(level string, logfile string) {
	logLevelMap := map[string]int {"critical":1, "error":2, "warn":3, "info":4, "debug":5, "trace":6}
	var logLevels []string
	if _, exists := logLevelMap[level]; exists {
		for k, v := range(logLevelMap) {
			if v <= logLevelMap[level] {
				logLevels = append(logLevels, k)
			}
		}
	} else {
		logLevels = append(logLevels, "info")
	}
	logConsole := ""
	if level == "trace" {
		logConsole = "<console/>"
	}
	fmt.Printf("init log, level=%s(%s) file=%s\n", level, logLevels, logfile)
	seeLogConfig := `<seelog>
	<outputs formatid="main">
		<filter levels="%s">
		<buffered size="10000" flushperiod="1000">
			<rollingfile type="date" filename="%s" datepattern="2006-01-02.15" maxrolls="120"/>
		</buffered>
		</filter>
		%s
    </outputs>
	<formats>
		<format id="main" format="[%%LEVEL]%%Date(2006-01-02 15:04:05.000000) %%Msg%%n"/>
	</formats>
	</seelog>`
	logConfig := fmt.Sprintf(seeLogConfig, strings.Join(logLevels, ","), logfile, logConsole)

	var err error
	DefaultLogger, err = log.LoggerFromConfigAsBytes([]byte(logConfig))
	if err != nil {
		fmt.Printf("init logger failed:%v\n", err)
	}

	log.ReplaceLogger(DefaultLogger)

	logConfig = fmt.Sprintf(seeLogConfig, "info", filepath.Dir(logfile) + "/monitor.log", logConsole)
	MonitorLogger, err = log.LoggerFromConfigAsBytes([]byte(logConfig))
	if err != nil {
		fmt.Printf("init monitor logger failed:%v\n", err)
	}
}

func WriteLog(level string, format string, params ...interface{}) {
	WriteLogWithGoID(level, 0, format, params...)
}

func WriteLogWithGoID(level string, goid uint64, format string, params ...interface{}) {
	if goid == 0 {
		goid = GetGoroutineID()
	}
	newParams := []interface{}{goid}
	newParams = append(newParams, params...)
	switch strings.ToLower(level) {
	case "debug":
		DefaultLogger.Debugf("[%d] " + format, newParams...)
	case "info":
		DefaultLogger.Infof("[%d] " + format, newParams...)
	case "warn":
		DefaultLogger.Warnf("[%d] " + format, newParams...)
	case "error":
		DefaultLogger.Errorf("[%d] " + format, newParams...)
		ErrorLogNumber++
	case "critical":
		DefaultLogger.Criticalf("[%d] " + format, newParams...)
		ErrorLogNumber++
	}
}

