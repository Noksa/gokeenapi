## Utility for adding/removing static routes in Keenetic routers via REST API

###### [Russian doc (Документация на русском)](https://github.com/Noksa/gokeenapi/blob/main/README_ru.md)

---

#### Limitations | Requirements
* No additional configuration is required on a router - just specify the router address in `gokeenapi` (for example, `http://192.168.1.1`)
* At the moment, `gokeenapi` can only work with a router in a local network (i.e. the device from which `gokeenapi` is launched and the router are in the same network)
---

#### What `gokeenapi` can already do:
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

#### Examples usage

The easiest way to start using `gokeenapi` is through docker containers

---

#### Docker

* Check all existing commands
```shell
export GOKEENAPI_IMAGE="noksa/gokeenapi:0.0.1"
docker pull "${GOKEENAPI_IMAGE}"
docker run --rm -ti "${GOKEENAPI_IMAGE}" --help
```

* View interfaces on the router - passing login\password\api via flags
```shell
export GOKEENAPI_IMAGE="noksa/gokeenapi:0.0.1"
docker pull "${GOKEENAPI_IMAGE}"
docker run --rm -ti "${GOKEENAPI_IMAGE}" show-interfaces --url http://192.168.1.1 --login admin --password admin
```

* View interfaces on the router - passing login\password\api via environment variables
```shell
export GOKEENAPI_IMAGE="noksa/gokeenapi:0.0.1"
docker run --rm -ti -e GOKEENAPI_URL="http://192.168.1.1" -e GOKEENAPI_LOGIN="admin" -e OKEENAPI_PASSWORD="admin" "${GOKEENAPI_IMAGE}" show-interfaces
```

* View interfaces on the router - passing login\password\api via a file with environment variables
```shell
export GOKEENAPI_IMAGE="noksa/gokeenapi:0.0.1"
touch .gokeenapienv
echo -e "GOKEENAPI_URL=http://192.168.1.1\n" >> .gokeenapienv
echo -e "GOKEENAPI_LOGIN=admin\n" >> .gokeenapienv
echo -e "GOKEENAPI_PASSWORD=admin\n" >> .gokeenapienv
docker run --rm -ti -v "$(pwd)/.gokeenapienv":"/gokeenapi/.gokeenapienv" "${GOKEENAPI_IMAGE}" show-interfaces
```

* View interfaces on the router - passing login\password\api via YAML config file
```shell
export GOKEENAPI_IMAGE="noksa/gokeenapi:0.0.1"
docker run --rm -ti -v "$(pwd)/config_example.yaml":"/gokeenapi/config.yaml" "${GOKEENAPI_IMAGE}" show-interfaces --config "/gokeenapi/config.yaml"
```