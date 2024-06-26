## Package `graceful` v1.1.2

### Default settings

**SIGINT** and **SIGTERM** are stopping the application
**SIGHUB** is gracefully restarting it.

Graceful shutdown:

* 1 minute to finish requests
* Then connections cut

Graceful restart:

* Socket file descriptor given to fork/exec child
* 30 minutes to finish requests for old process (can switch to shutdown with a
  SIGINT/SIGTERM on this process)


### Usage

```
graceful.NewService()
```

### Configuration with options

```
s := graceful.NewService(
	graceful.WithWaitDuration(30 * time.Second),
	graceful.WithReloadWaitDuration(time.Hour),
	graceful.WithPIDFile("/var/run/service.pid"),
)

err := s.ListenAndServe(ctx, "tcp", ":9000", handler)
```
