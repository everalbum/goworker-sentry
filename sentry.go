package sentry

import (
	"errors"
	"fmt"

	"github.com/getsentry/raven-go"
)

// Wrapper wraps goworkers and reports failed jobs to Sentry
func Wrapper(job string, w func(string, ...interface{}) error) func(string, ...interface{}) error {
	return func(queue string, args ...interface{}) error {
		defer func() {
			if e := recover(); e != nil { // capture panics
				capture(job, fmt.Sprint(e), args...)
			}
		}()

		err := w(queue, args...)

		if err != nil { // capture errors returned by job
			capture(job, err.Error(), args...)
		}
		return err
	}
}

func capture(job string, msg string, args ...interface{}) {
	packet := raven.NewPacket(msg, raven.NewException(errors.New(msg), raven.NewStacktrace(2, 3, raven.IncludePaths())))
	packet.Extra["Job"] = job
	packet.Extra["Arguments"] = args
	raven.Capture(packet, map[string]string{"job": job, "logger": "resque"})
}
