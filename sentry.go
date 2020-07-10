package sentry

import (
	"fmt"
	"github.com/getsentry/raven-go"
	"github.com/pkg/errors"
	"runtime"
)

// Wrapper wraps goworkers and reports failed jobs to Sentry
func Wrapper(job string, w func(string, ...interface{}) error) func(string, ...interface{}) error {
	return func(queue string, args ...interface{}) error {
		defer func() {
			if e := recover(); e != nil { // capture panics
				capture(job, fmt.Errorf("panic: %s", e), args...)
			}
		}()

		err := w(queue, args...)

		if err != nil { // capture errors returned by job
			capture(job, err, args...)
		}
		return err
	}
}

type stackTracer interface {
	StackTrace() errors.StackTrace
}

func getCauseWithStacktrace(err error) (stackTracer, bool) {
	type causer interface {
		Cause() error
	}
	if err == nil {
		return nil, false
	}
	// recursively go looking for the cause
	cause, cause_ok := err.(causer)
	if cause_ok {
		ret_err, ret_ok := getCauseWithStacktrace(cause.Cause())
		if ret_ok {
			return ret_err, ret_ok
		}
	}
	stack, stack_ok := err.(stackTracer)
	if stack_ok {
		return stack, stack_ok
	}
	return nil, false
}

// GetOrNewStacktrace tries to get stacktrace from err as an interface of github.com/pkg/errors, or else NewStacktrace()
func getOrNewStacktrace(err error, skip int, context int, appPackagePrefixes []string) *raven.Stacktrace {
	stacktrace, ok := getCauseWithStacktrace(err)
	if !ok {
		return raven.NewStacktrace(skip+1, context, appPackagePrefixes)
	}
	var frames []*raven.StacktraceFrame
	for _, f := range stacktrace.StackTrace() {
		pc := uintptr(f) - 1
		fn := runtime.FuncForPC(pc)
		var fName string
		var file string
		var line int
		if fn != nil {
			file, line = fn.FileLine(pc)
			fName = fn.Name()
		} else {
			file = "unknown"
			fName = "unknown"
		}
		frame := raven.NewStacktraceFrame(pc, fName, file, line, context, appPackagePrefixes)
		if frame != nil {
			frames = append([]*raven.StacktraceFrame{frame}, frames...)
		}
	}
	return &raven.Stacktrace{Frames: frames}
}

func capture(job string, err error, args ...interface{}) {
	packet := raven.NewPacket(err.Error(), raven.NewException(err, getOrNewStacktrace(err, 2, 3, raven.IncludePaths())))
	packet.Extra["Job"] = job
	packet.Extra["Arguments"] = args
	raven.Capture(packet, map[string]string{"job": job, "logger": "resque"})
}
