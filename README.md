# goworker-sentry
goworker wrapper that reports failed jobs to Sentry

## Usage
```go
goworker.Register("MyClass", sentry.Wrapper("MyClass", myWorker))
```
