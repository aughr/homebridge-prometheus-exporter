# Prometheus Exporter for Homebridge
A simple exporter for Prometheus that reads information about all your devices and exports all values as Prometheus metrics.

## Usage
`prometheus-exporter` is a command line program that takes few parameters:

```text
USAGE:
    homebridge-exporter [OPTIONS] --username <USERNAME> --password <PASSWORD>

OPTIONS:
        -debug                  Debug mode (displays additional log lines)
    -h, -help                   Print help information
        -keyfile <KEYFILE>      Authorization keys file. Default to authorization-keys.yml in the
                                 current working directory [default: authorization-keys.yml]
    -p, -password <PASSWORD>    Homebridge password
        -port <PORT>            Metrics webserver port (service /metrics for Prometheus scraper)
                                 [default: 9123]
        -prefix <PREFIX>        Registry metrics prefix [default: homebrige]
    -u, -username <USERNAME>    Homebridge username
        -uri <URI>              Homebridge UI uri [default: http://localhost:8581]
    -V, -version                Print version information
```
This software scrapes all the accessories from homebridge APIs and creates prometheus metrics out of all services information.
All the metrics are then exposed under the standard path `/metrics` by the embedded HTTP server.

## Build from source

```
go build
```

