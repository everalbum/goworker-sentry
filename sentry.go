package sentry

import "github.com/getsentry/raven-go"

// Wrapper wraps goworkers and reports failed jobs to Sentry
func Wrapper(job string, w func(string, ...interface{}) error) func(string, ...interface{}) error {
	return func(queue string, args ...interface{}) error {
		err := w(queue, args...)
		if err != nil {
			packet := raven.NewPacket(err.Error(), raven.NewException(err, raven.NewStacktrace(1, 3, nil)))
			packet.Extra["Job"] = job
			packet.Extra["Arguments"] = args
			raven.Capture(packet, map[string]string{"job": job, "logger": "resque"})
		}
		return err
	}
}
