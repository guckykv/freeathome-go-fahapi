# fahapi

Very first version of a GoLang library for accessing the local free@home API
of the [System Access Point 2.0 für Busch-free@home®](https://www.busch-jaeger.de/produktuebersicht?tx_nlbjproducts_catalog%5Baction%5D=show&tx_nlbjproducts_catalog%5BcatBjeProdukt%5D=42725&tx_nlbjproducts_catalog%5Bcontroller%5D=CatStdArtikel&cHash=8d65a7aae202e11a72f70d11ebc364d2)
(needs at least Access Point Software Version 2.6).

## fahapi - GoLang Package

This package reads in all devices from the System Access Point and connects via WebSocket to get all updates.
Some of the Device/Channel types are hydrated in easier usable Go Objects. 

Currently supported Device Types (FunctionIDs):
* FID_SWITCH_SENSOR                                  
* FID_DIMMING_SENSOR                                 
* FID_SWITCH_ACTUATOR                                
* FID_DIMMING_ACTUATOR                               
* FID_WINDOW_DOOR_SENSOR                             
* FID_ROOM_TEMPERATURE_CONTROLLER_MASTER_WITHOUT_FAN 
* FID_BRIGHTNESS_SENSOR                              
* FID_RAIN_SENSOR                                    
* FID_TEMPERATURE_SENSOR                             
* FID_WIND_SENSOR                                    

You can use a CallBack function to get a message for all updates (for the supported types).
Virtual devices can be created via `PutVirtualDevice`, and devices appearing while the
websocket loop runs are picked up automatically.

For examples how to use the package look into `fahinflux` and `fahcli`.

Create a client with `New`, load the model with `ReadAndHydrateAllDevices`, then run
`StartWebSocketLoop`. Cancel its context to shut down — the library installs no signal
handlers, that is the application's business. `TreatAllUnitsAsUpdated` forces a full
flush, e.g. on SIGHUP. The connection is kept alive with websocket pings and
re-established with a backoff when it drops.

## Example Usages of this package

Some example tools based on this package can be found [here](https://github.com/guckykv/freeathome-go-tools/)

### fahinflux - Writes all Updates for some Device Types into an InfluxDB

Writes all updates of all RTC, window sensors and weather station to InfluxDB.

See [fahinflux](https://github.com/guckykv/freeathome-go-tools/cmd/fahinflux).

### fahcli - Manage devices via shell command

Very first version of a shell command to make all sorts of operations possible via the f@h API.

See [fahcli](https://github.com/guckykv/freeathome-go-tools/cmd/fahcli).

### Limitations

* Works only with SysAP ID `00000000-0000-0000-0000-000000000000`. 
  So probably it doesn't work, if you have more than one SysAP.
  
* **Never send a text frame on the websocket.** A single text frame closes *every*
  websocket client of the SysAP, not just the sender, and its websocket service then
  needs a few seconds before it accepts new connections. Use ping frames, which the SysAP
  answers reliably. The number of clients is not a constraint — eight at once are served
  without trouble. Measured with
  [sysapprobe](https://github.com/guckykv/freeathome-go-tools/tree/main/cmd/sysapprobe).

* Reading a unit's fields from another goroutine is a data race. Read them in a callback,
  which runs in the websocket loop's goroutine. `Unit`, `Units`, `Device` and
  `Configuration` are safe from anywhere.

* No writing possibilities via the `UnitModel` data structure.
  If you want to change a value, you have to use `fahapi.PutDatapoint(sysapId, deviceId, channelId, datapointId, value)`.
  Via WebSocket connection the change will be synced into the `UnitModel` very quickly.
