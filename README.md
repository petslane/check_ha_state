# check_ha_state
A Nagios plugin for checking HomeAssistant sensor state.

This plugin connects to HomeAssistant API and checks that sensor state is not UNKNOWN or UNAVAILABLE.
Optionally checks sensor's last updated/changed state date.

## Build
```bash
$ go build 
```

Or use pre-built binaries from releases.

## Configuration

```bash
$ ./check_ha_state --help
Options:

  -h, --help               show help
      --url               *HomeAssistant url. Example: http://127.0.0.1:8123
  -e, --entity            *HomeAssistant entity id
      --token             *HomeAssistant API token
  -u, --last_updated_age   Maximum last updated age in seconds
  -c, --last_changed_age   Maximum last changed age in seconds
      --debug              Show debug info
```

## Usage

Simple check that sensor state would not be UNKNOWN or UNAVAILABLE:
```bash
$ ./check_ha_state -e sensor.outside_temperature --url http://localhost:8123/ --token super.secret.token
OK - sensor.outside_temperature | state=8.6 last_updated=2022-04-13T10:20:39.070113+00:00 last_changed=2022-04-13T10:20:39.070113+00:00
```

Check that sensor state last change was less than 60 seconds ago and last update less than 30 seconds ago:
```bash
$ ./check_ha_state -e sensor.outside_temperature --url http://localhost:8123/ --token super.secret.token -c 60 -u 30
CRITICAL - sensor.outside_temperature last update too long ago (290s > 30s)
```
