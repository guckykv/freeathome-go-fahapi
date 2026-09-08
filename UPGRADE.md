# Upgrading

This release changes the API. There were never any tags, so there is no version
to pin to the old shape — this document is the migration path.

The short version: the package no longer keeps its state in package variables.
You create a `Client` and call methods on it.

## Before and after

```go
// before
fahapi.ConfigureApi(host, user, pass, onUnits, onMessage, logger, 1)
fahapi.ReadAndHydradteAllDevices()
fahapi.StartWebSocketLoop(300)

unit := fahapi.UnitMap["ABB700D821D3.ch0000"]
```

```go
// after
api := fahapi.New(fahapi.Config{
    Host:            host,
    Username:        user,
    Password:        pass,
    UnitCallback:    onUnits,
    MessageCallback: onMessage,
    Logger:          logger,
    LogLevel:        1,
})

if err := api.ReadAndHydrateAllDevices(); err != nil {
    return err
}

ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

if err := api.StartWebSocketLoop(ctx, 300); err != nil {
    return err
}

unit := api.Unit("ABB700D821D3.ch0000")
```

## The changes in detail

### The import path is unchanged

The module moved from `github.com/guckykv/freeathome-go-fahapi/fahapi` to
`github.com/guckykv/freeathome-go-fahapi`, but since the package still lives in
the `fahapi/` subdirectory, the path you import stays exactly the same. Only the
`require` line in your `go.mod` changes.

### `ConfigureApi` → `New`

`ConfigureApi` took seven positional arguments. `New` takes a `Config` struct,
so adding an option later does not break callers again. `Logger` may be nil; it
defaults to the standard logger rather than panicking. `Timeout` is new and
bounds a single HTTP request (default 10s).

You may create more than one client now — the single hardcoded SysAP id is still
a limitation, but nothing else stops you from talking to two access points.

### The exported variables are gone

| before | after |
| --- | --- |
| `fahapi.UnitMap[key]` | `api.Unit(key)` — returns nil if unknown |
| `range fahapi.UnitMap` | `range api.Units()` — a snapshot of the map |
| `fahapi.FreeDevices[id]` | `api.Device(id)` |
| `fahapi.SysAPConfiguration` | `api.Configuration()` |

The accessors take a read lock, so they are safe from any goroutine. See
*Concurrency* below for what that does and does not cover.

### Everything that talked to the SysAP is a method

`GetConfiguration`, `GetDevice`, `GetDeviceList`, `GetDatapoint`, `PutDatapoint`
and `PutVirtualDevice` are methods on `Client`. The arguments are unchanged.

`GetFloorRoom` and `PrtAllUnits` are methods too.

### `ReadAndHydradteAllDevices` → `ReadAndHydrateAllDevices`

The old name was misspelled and it called `log.Fatal` when the configuration
could not be read, ending your process from inside the library. The new one
returns an error. The old spelling is gone along with the package-level API.

### `StartWebSocketLoop` takes a context, and reconnects

```go
StartWebSocketLoop(refreshTime int) error                      // before
StartWebSocketLoop(ctx context.Context, refreshTime int) error // after
```

Three behaviour changes come with it:

- **It no longer returns when the connection drops.** It reconnects with a
  backoff from 1s to 60s. A `nil` return now means the context was cancelled.
- **The library installs no signal handlers.** It used to call `signal.Notify`
  for SIGINT and SIGHUP, taking those signals away from the whole application.
  Cancel the context to shut down, and call `TreatAllUnitsAsUpdated(true)`
  yourself for the full flush that SIGHUP used to trigger.
- **The connection is kept alive with websocket pings** and a read deadline, so
  a half-open connection is detected instead of blocking forever.

### `log.Fatal` is gone from the library

A room temperature controller reporting an error, a malformed datapoint key in a
websocket message, and a failed configuration read each used to end the process.
They are logged now, or returned as an error. Handle them where it makes sense
for your application.

### Malformed values are rejected

Numeric conversions used to discard their error, so an empty or malformed value
became `0` and was reported as a genuine measurement. Such a datapoint now
produces no update at all. If you relied on receiving that `0`, you will see
nothing instead.

### New helpers

Almost every field of the API model is an optional pointer.

- `fahapi.Str(*string) string` dereferences one safely.
- `(*UnitData).DisplayName() string` is the channel's name, or `""`.

## Concurrency

The websocket loop applies every update from a single goroutine, and your
callbacks run in that goroutine. Inside a callback the model is consistent and
you may read unit fields freely. That is where reading belongs.

From another goroutine, use `Unit`, `Units`, `Device` and `Configuration`. Their
lock covers the maps, **not the fields of a unit** — the loop keeps writing
those, so reading them outside a callback is a data race. The `...Set` flags are
valid only for the duration of the callback that reported them.
