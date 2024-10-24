## Утилита для добавления\удаления статичных маршрутов в роутерах Кинетик через REST API

#### Видео

https://github.com/user-attachments/assets/404e89cc-4675-42c4-ae93-4a0955b06348

---

#### Важные примечания
* `gokeenapi` находится в процессе активной разработки!
* Дополнительной настройки на роутере не требуется - достаточно указать адрес роутера в конфигурации `gokeenapi` (например `http://192.168.1.1`)
* `gokeenapi` работает с роутерами Keenetic как по локальной сети так и по Интернету используя локальный адрес роутера (например `192.168.1.1`) или доменное имя от KeenDNS сервиса (например `my-router.keenetic.pro`)

---

#### Что утилита уже умеет:
* Выводить список интерфейсов которые уже добавлены в роутер - для удобного поиска ID интерфейса для которого нужно добавить\удалить маршруты
* Удалять статичные маршруты только для указанного интерфейса. В Веб-конфигураторе роутера на текущий момент можно только удалить все созданные статичные маршруты для всех интерфейсов сразу
* Добавлять\обновлять статичные маршруты для указанного интерфейса из bat файлов с диска
* Добавлять\обновлять статичные маршруты для указанного интерфейса из ссылок, которые ведут на bat файл (например [отсюда](https://iplist.opencck.org/?format=bat&data=cidr4&site=youtube.com))
---

#### Конфигурация

`gokeenapi` можно сконфигурировать несколькими путями:
* Через конфигурационный файл YAML
* Через переменные окружения
* Через файл с переменными окружениями которые надо загрузить
* Через флаги в командной строке

Все варианты можно совмещать - например логин\пароль и API URL можно хранить в переменных окружения, а список файлов откуда нужно добавить маршруты можно добавить в yaml конфиг файл или передать через флаги

---

#### Примеры конфигурации

* [Yaml файл](https://github.com/Noksa/gokeenapi/blob/main/config_example.yaml)
```yaml
keenetic:
  # IP адрес Keenetic
  url: http://192.168.1.1
  # Логин юзера который может взаимодействовать с REST API - обычно это админ
  login: super-login
  # Пароль от юзера
  # Предпочтительнее хранить пароль через переменные окружения или передавать через флаг командной строки при отключенной истории оболочки
  password: super-password
  interface:
    # ID интерфейса на роутере для которого надо удалять\добавлять маршруты
    # Найти этот ID легко через команду show-interfaces после того как вы добавили ВПН подключение
    id: "Wireguard0"

# Бат-файлы из которых следует загрузить маршруты в роутер
# Данные файлы должны быть размещены на дисках
bat-file:
  - /path/to/batfile.bat

# Бат-файлы из ссылок из которых следует загрузить маршруты в роутер
# Ссылка должна вести на файл в формате BAT
bat-url:
  - https://iplist.opencck.org/?format=bat&data=cidr4&site=instagram.com
  - https://iplist.opencck.org/?format=bat&data=cidr4&site=youtube.com
  - https://iplist.opencck.org/?format=bat&data=cidr4&site=facebook.com
```

* Переменные окружения - могут быть экспортированы в оболочку любым удобным способом (.bashrc/.zshrc и так далее)
```shell
export GOKEENAPI_URL="http://192.168.1.1"
export GOKEENAPI_LOGIN="admin"
export GOKEENAPI_PASSWORD="password"
./gokeenapi ...
```

* Файл с переменными окружения (`.gokeenapienv`) - должен лежать рядом с запускаемым файлом `gokeenapi`.

    Содержимое файла:
```shell
GOKEENAPI_LOGIN=admin
GOKEENAPI_URL=http://192.168.1.1
GOKEENAPI_PASSWORD=password
```

* Через флаги командной строки
```shell
./gokeenapi --url http://192.168.1.1 --login admin --password password
```

---

#### Примеры использования

Самый простой способ начать пользоваться `gokeenapi` через docker контейнеры

---

#### Docker 

* Посмотреть интерфейсы на роутере - передача логин\пароля\апи через флаги
```shell
export GOKEENAPI_IMAGE="noksa/gokeenapi:0.0.1"
docker pull "${GOKEENAPI_IMAGE}"
docker run --rm -ti "${GOKEENAPI_IMAGE}" show-interfaces --url http://192.168.1.1 --login admin --password admin
```

* Удалить все созданные маршруты для указанного интерфейса на роутере - передача логин\пароля\апи через переменные окружения
* Обратите внимание, что флаг `--interface-id` обязателен (он так же может быть передан как переменная окружения либо через yaml конфиг файл)
```shell
export GOKEENAPI_IMAGE="noksa/gokeenapi:0.0.1"
docker run --rm -ti -e GOKEENAPI_URL="http://192.168.1.1" -e GOKEENAPI_LOGIN="admin" -e OKEENAPI_PASSWORD="admin" "${GOKEENAPI_IMAGE}" delete-routes --interface-id "Wireguard0"
```

* Посмотреть интерфейсы на роутере - передача логин\пароля\апи через файл с переменными окружениями
```shell
export GOKEENAPI_IMAGE="noksa/gokeenapi:0.0.1"
touch .gokeenapienv
echo -e "GOKEENAPI_URL=http://192.168.1.1\n" >> .gokeenapienv
echo -e "GOKEENAPI_LOGIN=admin\n" >> .gokeenapienv
echo -e "GOKEENAPI_PASSWORD=admin\n" >> .gokeenapienv
docker run --rm -ti -v "$(pwd)/.gokeenapienv":"/gokeenapi/.gokeenapienv" "${GOKEENAPI_IMAGE}" show-interfaces
```

* Добавить маршруты для интерфейса на роутере - передача логин\пароля\апи через YAML конфиг файл
* Передача `--interface-id` через флаг
* Передача списков `bat-file` и `bat-url` через yaml конфиг (можно так же через флаг) 
```shell
export GOKEENAPI_IMAGE="noksa/gokeenapi:0.0.1"
docker run --rm -ti -v "$(pwd)/config_example.yaml":"/gokeenapi/config.yaml" "${GOKEENAPI_IMAGE}" add-routes --config "/gokeenapi/config.yaml" --interface-id "Wireguard0"
```
