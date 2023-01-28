# No Maintenance Intended

[![No Maintenance Intended](http://unmaintained.tech/badge.svg)](http://unmaintained.tech/)

# Credits

Forked from https://github.com/szatmary/sonos.

# Changes

There are few changes have been made mostly around the structure of the generated code and how it is consumed.
There are also improvements to the event lifecycle (subscribe/renew/unsubscribe) with some other miscellaneous changes.

# Services

The service implimentations are automatically generated from the service definition XML files obtained from the Sonos devices via `makeservice.go.`

`cmd/makeservices/downloadallservices.sh` fetches them from the device and `cmd/makeservices/makeallservices.sh` generates the code.

# More

Please see https://svrooij.io/sonos-api-docs/sonos-communication.html and https://svrooij.io/sonos-api-docs/services/ for Sonos API and http://upnp.org/ for UPnP.

