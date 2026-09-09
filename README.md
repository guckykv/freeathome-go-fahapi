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

For examples how to use the package look into `fahinflux` and `fahcli`.

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
  
* ~~VirtualDevices not yet implemented.~~
  PUT call for creating virtual devices is implemented. And the standard Unit logging now shows the NativeId too.
  The fhapi also supports, that new devices show up while the websocket loop already runs.

* **Never send a text frame on the websocket.** Measured against a System Access Point
  (software 2.6): a single text frame closes not just the sender's connection but *every*
  websocket client connected to the SysAP, and its websocket service then needs a few
  seconds before it accepts new connections. Use ping frames as a keepalive, which the
  SysAP answers reliably. Eight simultaneous clients were served without trouble, so the
  number of connections is not the constraint -- the frame type is.

* `StartWebSocketLoop` takes a `context.Context` as its first argument. Cancel it to shut
  down; the library no longer installs signal handlers of its own. Use
  `fahapi.TreatAllUnitsAsUpdated(true)` for the full flush that used to be bound to SIGHUP.
  The connection is kept alive with websocket pings and re-established with a backoff when
  it drops.

* No writing possibilities via the `UnitModel` data structure.
  If you want to change a value, you have to use `fahapi.PutDatapoint(sysapId, deviceId, channelId, datapointId, value)`.
  Via WebSocket connection the change will be synced into the `UnitModel` very quickly.
