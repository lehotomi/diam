package mlog

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	//"fmt"
)

var (
	Trace *log.Logger
	Info  *log.Logger
	Warn  *log.Logger
	Error *log.Logger
)

//var logFlags int =  log.Ldate|log.Ltime|log.Lshortfile
var inited bool = false
var env map[string]string = make(map[string]string)

var discard io.Writer = ioutil.Discard

const default_flag = log.Lshortfile | log.Ltime

func init() {    
	Trace = log.New(discard, "TRACE ", default_flag)
	Info = log.New(discard, "INFO ", default_flag)
	Warn = log.New(os.Stderr, "WARN ", default_flag)
	Error = log.New(os.Stderr, "ERROR ", default_flag)
}

func Init(
	traceHandle io.Writer,
	infoHandle io.Writer,
	warningHandle io.Writer,
	errorHandle io.Writer,
	flags int) {

	if inited {
		return
	}

	inited = true

	for _, e := range os.Environ() {
		if i := strings.Index(e, "="); i >= 0 {
			env[e[:i]] = e[i+1:]
		}
	}

	c_log, ok := env["LOG_LEVEL"]

	if !ok {
		Trace = log.New(discard, "TRACE ", flags)
		Info = log.New(discard, "INFO ", flags)
		Warn = log.New(warningHandle, "WARN ", flags)
		Error = log.New(errorHandle, "ERROR ", flags)
		return
	}

	if c_log == "TRACE" {
		Trace = log.New(traceHandle, "TRACE ", flags)
		Info = log.New(infoHandle, "INFO ", flags)
		Warn = log.New(warningHandle, "WARN ", flags)
		Error = log.New(errorHandle, "ERROR ", flags)
		return
	}

	if c_log == "INFO" {
		Trace = log.New(discard, "TRACE ", flags)
		Info = log.New(infoHandle, "INFO ", flags)
		Warn = log.New(warningHandle, "WARN ", flags)
		Error = log.New(errorHandle, "ERROR ", flags)
		return
	}

	if c_log == "WARNING" {
		Trace = log.New(discard, "TRACE ", flags)
		Info = log.New(discard, "INFO ", flags)
		Warn = log.New(warningHandle, "WARN ", flags)
		Error = log.New(errorHandle, "ERROR ", flags)
		return
	}

	if c_log == "ERROR" {
		Trace = log.New(discard, "TRACE ", flags)
		Info = log.New(discard, "INFO ", flags)
		Warn = log.New(discard, "WARN ", flags)
		Error = log.New(errorHandle, "ERROR ", flags)
		return
	}

	Trace = log.New(discard, "TRACE: ", flags)
	Info = log.New(discard, "INFO: ", flags)
	Warn = log.New(discard, "WARN: ", flags)
	Error = log.New(discard, "ERROR: ", flags)

	//Info.Println("Env:", env["LOG_LEVEL"])
}
