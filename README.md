## Utility to run commands (such add add/del routes) in Keenetic routers via REST API

#### Video

https://github.com/user-attachments/assets/404e89cc-4675-42c4-ae93-4a0955b06348

---

#### Version with UI

There is a GUI `gokeenapi` version available [here](https://github.com/Noksa/gokeenapiui)

If you don't like or don't know how to use CLI programs, consider using the GUI version

---

#### Important notes
* No additional configuration is required on a router - just specify the router address in `gokeenapi` (for example, `http://192.168.1.1`)
* `gokeenapi` works with Keenetic routers over LAN or Internet using internal router IP address (like `192.168.1.1`) or domain from KeenDNS (like `my-router.keenetic.pro`)
---

#### What `gokeenapi` can already do:
* Apply ASC parameters to existing WG connections from WG conf files
* Display a list of interfaces that have already been added to the router - for easy search of the interface ID for which you need to add/remove routes
* Delete static routes only for the specified interface. In the Web Configurator of the router, at the moment you can only delete all created static routes for all interfaces at once
* Add\update static routes for the specified interface from bat files from disk
* Add\update static routes for the specified interface from links that download bat file (for example [from here](https://iplist.opencck.org/?format=bat&data=cidr4&site=youtube.com))
---

#### Configuration

`gokeenapi` can be configured in several ways:
* Through a YAML configuration file
* Through environment variables
* Through a file with environment variables that need to be loaded
* Through flags in the command line

All options can be combined - for example, login\password and Router URL can be stored in environment variables, and the list of files from which you need to add routes can be added to the yaml config file or passed through flags

---

#### Videos

Videos with examples (note that language is **Russian**:
* [Routes](https://www.youtube.com/watch?v=lKX74btFypY)

---

#### Examples

The easiest way to start using `gokeenapi` is through docker containers or using the latest available release from [here](https://github.com/Noksa/gokeenapi/releases)

---

#### Docker

It is recommended to use `noksa/gokeenapi:stable` image

* Check all existing commands
```shell
export GOKEENAPI_IMAGE="noksa/gokeenapi:stable"
docker pull "${GOKEENAPI_IMAGE}"
docker run --rm -ti "${GOKEENAPI_IMAGE}" --help
```

* View Wireguard interfaces on the router - passing login\password\api via flags
```shell
export GOKEENAPI_IMAGE="noksa/gokeenapi:stable"
docker pull "${GOKEENAPI_IMAGE}"
docker run --rm -ti "${GOKEENAPI_IMAGE}" show-interfaces --url http://192.168.1.1 --login admin --password admin --type Wireguard
```

* View interfaces on the router - passing login\password\api via environment variables
```shell
export GOKEENAPI_IMAGE="noksa/gokeenapi:stable"
docker run --rm -ti -e GOKEENAPI_URL="http://192.168.1.1" -e GOKEENAPI_LOGIN="admin" -e OKEENAPI_PASSWORD="admin" "${GOKEENAPI_IMAGE}" show-interfaces
```

* View interfaces on the router - passing login\password\api via a file with environment variables
```shell
export GOKEENAPI_IMAGE="noksa/gokeenapi:stable"
touch .gokeenapienv
echo -e "GOKEENAPI_URL=http://192.168.1.1\n" >> .gokeenapienv
echo -e "GOKEENAPI_LOGIN=admin\n" >> .gokeenapienv
echo -e "GOKEENAPI_PASSWORD=admin\n" >> .gokeenapienv
docker run --rm -ti -v "$(pwd)/.gokeenapienv":"/gokeenapi/.gokeenapienv" "${GOKEENAPI_IMAGE}" show-interfaces
```

* View interfaces on the router - passing login\password\api via YAML config file
```shell
export GOKEENAPI_IMAGE="noksa/gokeenapi:stable"
docker run --rm -ti -v "$(pwd)/config_example.yaml":"/gokeenapi/config.yaml" "${GOKEENAPI_IMAGE}" show-interfaces --config "/gokeenapi/config.yaml"
```
