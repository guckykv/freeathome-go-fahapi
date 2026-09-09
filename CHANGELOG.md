# Changelog

## Fixed

* The `Authorization` header was sent as `Basic: <credentials>`. Current SysAP firmware
  answers **401 to every request**, so the library did not work at all.
* The websocket keepalive sent a text frame every second. A text frame disconnects *every*
  websocket client of the SysAP, the free@home app included. It sends ping frames now.
* A lost connection ended the loop, which returned `nil` — indistinguishable from a clean
  shutdown. And with no read deadline, a half-open link blocked forever without any error.
* `log.Fatal` in library code ended the host process: on a failed configuration read, on a
  malformed datapoint key, and whenever a room temperature controller reported an error.
* Nil dereferences on optional API fields — a channel without `functionID`, a device with
  a floor but no room, a missing display name, an empty datapoint list.
* Unparsable numbers were silently recorded as `0` and passed on as real measurements.
* HTTP requests had no timeout, never closed their response body, and built a new client
  each time.

## Added

* Reconnect with a backoff of 1s to 60s, ping keepalive and a read deadline, so a dead
  link is detected and recovered from.
* Several clients per process, and thereby testability: `New(Config{...})`.
* `Str`, `(*UnitData).DisplayName`, `TreatAllUnitsAsUpdated`.
* Test suite, GitHub Actions, and a `Makefile` — `make check` runs gofmt, vet, build and
  `test -race`.

## Changed

* One module at the repository root. **The import path is unchanged**; only the `require`
  line moves from `…/freeathome-go-fahapi/fahapi` to `…/freeathome-go-fahapi`.
* Package-level state replaced by a `Client`.
* `StartWebSocketLoop` takes a `context.Context`; the library installs no signal handlers.

# Upgrading

```go
// before
fahapi.ConfigureApi(host, user, pass, onUnits, onMessage, logger, 1)
fahapi.ReadAndHydradteAllDevices()
fahapi.StartWebSocketLoop(300)
unit := fahapi.UnitMap[key]

// after
api := fahapi.New(fahapi.Config{Host: host, Username: user, Password: pass,
    UnitCallback: onUnits, MessageCallback: onMessage, Logger: logger, LogLevel: 1})
if err := api.ReadAndHydrateAllDevices(); err != nil { return err }

ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()
if err := api.StartWebSocketLoop(ctx, 300); err != nil { return err }
unit := api.Unit(key)
```

| Before | After |
| --- | --- |
| `ConfigureApi(...)` | `New(Config{...})` |
| `ReadAndHydradteAllDevices()` | `api.ReadAndHydrateAllDevices() error` |
| `StartWebSocketLoop(n)` | `api.StartWebSocketLoop(ctx, n)` |
| `fahapi.UnitMap[key]` | `api.Unit(key)` |
| `range fahapi.UnitMap` | `range api.Units()` |
| `fahapi.FreeDevices[id]` | `api.Device(id)` |
| `fahapi.SysAPConfiguration` | `api.Configuration()` |
| `fahapi.GetDevice(...)` and every other API call | method on `api` |

Three changes no compiler will point out:

* **Shutdown is yours.** Cancel the context; SIGINT and SIGHUP are no longer taken by the
  library. Call `TreatAllUnitsAsUpdated(true)` for the full flush SIGHUP used to trigger.
* **The loop no longer returns on connection loss**, it reconnects. `nil` means the
  context was cancelled.
* **A malformed value produces no update at all**, where it used to produce a `0`.

Reading a unit's fields from another goroutine is a data race — do it in a callback, which
runs in the loop's goroutine. `Unit`, `Units`, `Device` and `Configuration` are safe from
anywhere.
