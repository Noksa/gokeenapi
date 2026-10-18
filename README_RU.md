<div align="center">

<img src="logo.png" alt="gokeenapi logo" width="512">

# 🚀 gokeenapi

**Автоматизируйте управление роутером Keenetic (Netcraze) с легкостью**

<p align="center">
  <video src="https://github.com/user-attachments/assets/404e89cc-4675-42c4-ae93-4a0955b06348" width="100%"></video>
</p>

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Docker Pulls](https://img.shields.io/docker/pulls/noksa/gokeenapi)](https://hub.docker.com/r/noksa/gokeenapi)
[![GitHub release](https://img.shields.io/github/release/Noksa/gokeenapi.svg)](https://github.com/Noksa/gokeenapi/releases)

*Устали кликать по веб-интерфейсу Keenetic (Netcraze)? Автоматизируйте управление роутером с помощью простых CLI команд.*

<div align="center">

### [🇬🇧 **English documentation** 🇬🇧](README.md)

</div>

[🚀 Быстрый старт](#-быстрый-старт) • [📖 Документация](#-команды) • [📋 Справочник конфигурации](docs/config-reference-ru.md) • [🎨 GUI версия](https://github.com/Noksa/gokeenapiui) • [🤝 Участие в разработке](#-участие-в-разработке)

</div>

---

## ✨ О проекте

`gokeenapi` — CLI-инструмент для автоматизации управления роутерами Keenetic (Netcraze). Поддерживает управление маршрутами, DNS-записями, DNS-маршрутизацией, WireGuard-соединениями, известными хостами и планировщиком задач — через YAML-конфиг, без изменений настроек роутера. Работает по LAN или удалённо через KeenDNS.

---

## 🚀 Быстрый старт

Самый простой способ начать - использовать Docker или скачать последний релиз.

### 🐳 Docker (Рекомендуется)

Использование Docker — рекомендуемый способ запуска `gokeenapi`.

На [Docker Hub](https://hub.docker.com/r/noksa/gokeenapi) доступны два варианта тегов:

| Тег | Описание |
|-----|----------|
| `stable` | Последний стабильный релиз (обновляется при каждом релизе) |
| `1.7.0`, `1.7.1`, … | Конкретная версия — доступно начиная с `v1.7.0` |

```bash
# Скачиваем последний стабильный образ
export GOKEENAPI_IMAGE="noksa/gokeenapi:stable"
# Или зафиксировать конкретную версию (доступно начиная с v1.7.0)
# export GOKEENAPI_IMAGE="noksa/gokeenapi:1.7.0"
docker pull "${GOKEENAPI_IMAGE}"

# Запуск команды. Без --config образ читает /etc/gokeenapi/config.yaml.
docker run --rm -ti \
  -v "$(pwd)/config_example.yaml":/etc/gokeenapi/config.yaml \
  "${GOKEENAPI_IMAGE}" show-interfaces
```

Если `--config` не указан, образ берёт конфиг из `/etc/gokeenapi/config.yaml`. Свой путь задаётся флагом `--config` или переменной `GOKEENAPI_CONFIG` при запуске.

`/etc/gokeenapi/.gokeenapi` — каталог cookie сессии, он должен быть доступен на запись. Если смонтировать только файл конфига, каталог останется в контейнере. Если смонтировать каталог на `/etc/gokeenapi`, он закроет каталог из образа, поэтому такое монтирование должно быть с правом записи. Процесс работает от пользователя `gokeenapi`. На Linux файлу обычно ставят права `0644`, чтобы этот пользователь мог его прочитать, а пароль передают через `GOKEENAPI_KEENETIC_PASSWORD`.

### 📦 Последний релиз

Скачайте последний релиз для вашей платформы:

<div align="center">

[![Download Latest](https://img.shields.io/badge/📦_Скачать-Последний_релиз-green?style=for-the-badge)](https://github.com/Noksa/gokeenapi/releases)

</div>

---

## ⚙️ Конфигурация

`gokeenapi` настраивается с помощью `yaml` файла. Пример можно найти [здесь](https://github.com/Noksa/gokeenapi/blob/main/config_example.yaml). Полное описание каждого поля смотрите в [Справочнике конфигурации](docs/config-reference-ru.md).

Чтобы использовать свой файл конфигурации, передайте флаг `--config <путь>`. В Docker-образе этот путь уже задан: `/etc/gokeenapi/config.yaml`.

### Переиспользуемые списки Bat-файлов и Bat-URL

При управлении несколькими роутерами с одинаковой конфигурацией маршрутизации, вы можете создать общий YAML файл с путями к `bat-file`, путями к `bat-url` или и теми и другими, и ссылаться на него в разных конфигах.

**batfiles/common.yaml:**
```yaml
bat-file:
  - /path/to/discord.bat
  - /path/to/youtube.bat
bat-url:
  - https://example.com/instagram.bat
  - https://example.com/extra.bat
```

**Конфиг роутера:**
```yaml
routes:
  - interfaceId: Wireguard0
    bat-file:
      - batfiles/common.yaml         # Раскрывается: используются только bat-file записи
      - /path/to/router-specific.bat # Можно смешивать с обычными путями
    bat-url:
      - batfiles/common.yaml         # Раскрывается: используются только bat-url записи
      - https://example.com/other.bat
```

Утилита автоматически определяет `.yaml`/`.yml` файлы в массивах `bat-file` и `bat-url` и раскрывает их в соответствующие записи списка. Когда YAML файл указан в `bat-file`, используется только его список `bat-file`; когда указан в `bat-url`, используется только его список `bat-url`. Относительные пути в YAML-файлах списков разрешаются относительно директории YAML-файла.

### Переменные окружения

Все параметры конфигурации можно задать через переменные окружения:

| Переменная | Описание |
|---|---|
| `GOKEENAPI_CONFIG` | Путь к файлу конфигурации (альтернатива флагу `--config`). В официальном образе задан `/etc/gokeenapi/config.yaml` |
| `GOKEENAPI_KEENETIC_LOGIN` | Логин администратора роутера |
| `GOKEENAPI_KEENETIC_PASSWORD` | Пароль администратора роутера |
| `GOKEENAPI_INSIDE_DOCKER` | Если задана, использует `/etc/gokeenapi` в качестве директории данных |

`GOKEENAPI_KEENETIC_LOGIN` и `GOKEENAPI_KEENETIC_PASSWORD` позволяют не хранить чувствительные данные в конфигурационном файле. Официальный образ сам задаёт `GOKEENAPI_INSIDE_DOCKER` и `GOKEENAPI_CONFIG=/etc/gokeenapi/config.yaml`.

> **Рекомендация по безопасности**: Храните учётные данные в переменных окружения, а не в конфигурационном файле. При запуске программа предупредит, если файл конфигурации доступен для чтения группой или другими пользователями (права `0644` и шире). Ограничьте права командой `chmod 600 config.yaml` и передавайте логин/пароль через `GOKEENAPI_KEENETIC_LOGIN` / `GOKEENAPI_KEENETIC_PASSWORD`. Добавьте `config.yaml` и `config_*.yaml` в `.gitignore`, чтобы не допустить случайного попадания в репозиторий (`.gitignore` проекта уже содержит эти паттерны).

### Проверка TLS-сертификата

При подключении к роутеру по HTTPS с самоподписанным сертификатом, установите `tls_skip_verify: true` в секции `keenetic`:

```yaml
keenetic:
  url: https://192.168.1.1
  login: admin
  password: secret
  tls_skip_verify: true  # Отключить проверку TLS для самоподписанных сертификатов
```

> **Примечание**: Используйте `tls_skip_verify` только в доверенных локальных сетях. Отключение проверки сертификата делает соединение с роутером уязвимым к атакам типа «человек посередине». Для внешних URL со списками доменов по-прежнему требуется действительный TLS-сертификат.

---

## 📋 Справочник конфигурации

Полный справочник по всем полям файла `config.yaml` смотрите в **[docs/config-reference-ru.md](docs/config-reference-ru.md)**.

Смотрите также [config_example.yaml](config_example.yaml) для полностью аннотированного примера.

---

## 🔧 Поддерживаемые роутеры

`gokeenapi` протестирован со следующими моделями роутеров Keenetic (Netcraze):

- **Keenetic (Netcraze) Start**
- **Keenetic (Netcraze) Viva** 
- **Keenetic (Netcraze) Giga**

Поскольку утилита работает с Keenetic (Netcraze) Start (самая доступная модель в линейке), она должна быть совместима со всеми моделями роутеров Keenetic (Netcraze).

---

## 🎬 Видео демонстрации

Посмотрите эти видео демонстрации, чтобы увидеть `gokeenapi` в действии:

*   [Управление маршрутами](https://www.youtube.com/watch?v=lKX74btFypY)

---

## 🕐 Scheduler - Автоматизированное выполнение задач

Планировщик позволяет автоматизировать управление роутером, выполняя задачи с заданными интервалами или в фиксированное время. Идеально подходит для автоматического обновления маршрутов и DNS записей.

### Ключевые возможности

- **Выполнение по интервалу**: Запуск задач каждые N часов/минут (например, каждые 3 часа)
- **Выполнение по времени**: Запуск задач в определенное время (например, в 02:00, 06:00, 12:00)
- **Цепочка команд**: Последовательное выполнение нескольких команд (например, delete-routes → add-routes)
- **Поддержка нескольких роутеров**: Управление несколькими роутерами одной задачей
- **Механизм повтора**: Автоматический повтор неудачных задач с настраиваемой задержкой
- **Последовательное выполнение**: Задачи выполняются в очереди для избежания конфликтов

При остановке планировщика ожидание повторных попыток отменяется, а постановка новых задач безопасно завершается.

### Быстрый старт

```shell
# Запустить планировщик с конфигом
./gokeenapi scheduler --config scheduler.yaml
```

### Пример конфигурации

```yaml
tasks:
  - name: "Обновление маршрутов каждые 3 часа"
    commands:
      - add-routes
    configs:
      - /path/to/router1.yaml
      - /path/to/router2.yaml
      - /path/to/router3.yaml
    interval: "3h"
  
  - name: "Обновление маршрутов ежедневно с повтором"
    commands:
      - delete-routes
      - add-routes
    configs:
      - /path/to/router1.yaml
    times:
      - "02:00"
    retry: 3           # Повторить до 3 раз при ошибке
    retryDelay: "30s"  # Ждать 30 секунд между попытками
```

📖 **[Полная документация Scheduler →](SCHEDULER_RU.md)**

См. также: [scheduler_example.yaml](scheduler_example.yaml)

---

### 📚 Команды

Вот некоторые вещи, которые вы можете делать с `gokeenapi`. Для полного списка команд и опций используйте флаг `--help`.

```shell
./gokeenapi --help
```

#### `show-interfaces`

*Псевдонимы: `showinterfaces`, `si`, `showinterface`, `show-interface`*

Отображает все доступные интерфейсы на вашем роутере Keenetic (Netcraze).

```shell
# Показать все интерфейсы
./gokeenapi show-interfaces --config my_config.yaml

# Показать только WireGuard интерфейсы
./gokeenapi show-interfaces --config my_config.yaml --type Wireguard
```

#### `add-routes`

*Псевдонимы: `addroutes`, `ar`*

Добавляет статические маршруты в ваш роутер.

```shell
./gokeenapi add-routes --config my_config.yaml
```

#### `delete-routes`

*Псевдонимы: `deleteroutes`, `dr`*

Удаляет статические маршруты для конкретного интерфейса.

```shell
# Удалить маршруты для всех интерфейсов в конфигурационном файле
./gokeenapi delete-routes --config my_config.yaml

# Удалить маршруты для конкретного интерфейса
./gokeenapi delete-routes --config my_config.yaml --interface-id <ваш-interface-id>

# Удалить маршруты без подтверждения
./gokeenapi delete-routes --config my_config.yaml --force
```

> **Совет:** Чтобы найти ID интерфейсов, выполните команду `show-interfaces`.

#### `delete-all-routes`

*Псевдонимы: `deleteallroutes`, `dar`*

Удаляет все статические маршруты с роутера одним запросом, независимо от интерфейса.

```shell
# Удалить все маршруты (с подтверждением)
./gokeenapi delete-all-routes --config my_config.yaml

# Удалить все маршруты без подтверждения
./gokeenapi delete-all-routes --config my_config.yaml --force
```

> **Внимание:** Эта команда удаляет ВСЕ пользовательские статические маршруты на роутере. Используйте с осторожностью.

#### `add-dns-records`

*Псевдонимы: `adddnsrecords`, `adr`*

Добавляет статические DNS записи.

```shell
./gokeenapi add-dns-records --config my_config.yaml
```

#### `delete-dns-records`

*Псевдонимы: `deletednsrecords`, `ddr`*

Удаляет статические DNS записи на основе вашего конфигурационного файла.

```shell
./gokeenapi delete-dns-records --config my_config.yaml
```

#### `add-dns-routing`

*Псевдонимы: `adddnsrouting`, `adnsr`, `adddnsroutes`, `add-dns-routes`*

Добавляет правила DNS-маршрутизации (маршрутизация по доменам) в ваш роутер. Эта функция позволяет направлять трафик для определенных доменов через указанные сетевые интерфейсы.

**Требования:** Прошивка Keenetic версии 5.0.1 или выше

```shell
./gokeenapi add-dns-routing --config my_config.yaml
```

**Как это работает:**
- Загружает домены из локальных .txt файлов и удаленных URL
- Создает группы доменов (object-groups), содержащие указанные домены, IPv4-адреса и IPv4-подсети
- Связывает каждую группу с сетевым интерфейсом через dns-proxy маршруты
- Трафик для доменов в группе автоматически направляется через указанный интерфейс

Группы с более чем 300 уникальными записями автоматически делятся после сортировки: 397 доменов `facebook` станут группами `facebook` (300) и `facebook-2` (97). Следующие запуски обновляют части, переносят их маршруты при изменении `interfaceId` и удаляют ненужные дополнительные группы. Пустой источник оставляет существующие группы без изменений. Не используйте имена создаваемых частей, например `facebook-2`, в конфигурации.

Доменные записи приводятся к строчным ASCII-именам (IDNA); завершающие точки и повторяющиеся записи удаляются. IPv4-подсети приводятся к адресу сети: `10.0.0.1/24` становится `10.0.0.0/24`. Правила `full:` и `domain:` поддерживаются. Правила `regexp:`, `keyword:` и `include:` пропускаются, поскольку роутер не может представить их как обычные домены. Непустой источник, содержащий только неподдерживаемые или неверные записи, вызывает ошибку. Имя группы не может содержать пробелы или разделители команд роутера. Если для одной группы найдены два dns-proxy маршрута, обновление останавливается для проверки неоднозначного состояния роутера. При ошибке пакета API команда останавливается перед следующими пакетами и сохранением; команды, уже принятые в этом пакете, могут остаться в текущей конфигурации роутера.

**Источники доменов:**
- Локальные .txt файлы с одним доменом, IPv4-адресом или IPv4-подсетью на строку (поддерживаются комментарии с #)
- Удалённые URL со списками такого же формата
- YAML файлы, содержащие списки путей к domain-file или domain-url (для организации)

**Раскрытие YAML:** Утилита автоматически определяет `.yaml`/`.yml` файлы в массивах `domain-file` и `domain-url` и раскрывает их в содержащиеся в них пути к доменам (аналогично раскрытию bat-file/bat-url).

> **Важное правило:** когда `.yaml` файл указан в `domain-file` — из него извлекается только список `domain-file`; когда указан в `domain-url` — только список `domain-url`. Чтобы использовать и локальные файлы, и удалённые URL из одного YAML-манифеста, нужно указать его **в обоих** ключах:
>
> ```yaml
> # domains/social.yaml
> domain-file:
>   - social.txt                              # локальные домены
> domain-url:
>   - https://example.com/social-extra.txt    # удалённые домены
> ```
>
> ```yaml
> # Группа в конфиге роутера
> - name: social
>   domain-file:
>     - domains/social.yaml   # извлекает domain-file → social.txt
>   domain-url:
>     - domains/social.yaml   # извлекает domain-url → удалённый URL
>   interfaceId: Wireguard0
> ```
>
> Если указать YAML только в `domain-file` — удалённые URL внутри него будут молча проигнорированы, и наоборот.

**НОВОЕ: Переиспользуемые группы DNS-маршрутизации**

Теперь вы можете создавать общие YAML файлы с полными определениями групп DNS-маршрутизации и импортировать их в конфигурации нескольких роутеров. Это отличается от раскрытия domain-file/domain-url - вы импортируете целые определения групп, а не только списки доменов.

**custom/common_dns_groups.yaml:**
```yaml
groups:
  - name: youtube
    domain-url:
      - domains/youtube.yaml
    interfaceId: Wireguard0
  - name: telegram
    domain-url:
      - domains/telegram.yaml
    interfaceId: Wireguard0
  - name: trackers
    domain-file:
      - domains/trackers.yaml
    interfaceId: Wireguard0
```

**Конфиг роутера:**
```yaml
dns:
  routes:
    groups:
      - common_dns_groups.yaml    # Импорт всех групп из файла
      - name: router-specific     # Смешивание с группами конкретного роутера
        domain-file:
          - domains/local.txt
        interfaceId: GigabitEthernet0
```

Это позволяет хранить общие правила DNS-маршрутизации в одном месте и использовать их на всех ваших роутерах. Когда вы добавляете telegram в один роутер, просто обновите `common_dns_groups.yaml` и все роутеры, использующие его, получат обновление.

Существует два способа импортировать файл с общими группами:

```yaml
groups:
  # Строка — импорт всех групп как есть
  - common_dns_groups.yaml

  # Объект с ключом file: — тот же импорт, но с переопределением interfaceId
  # Все группы из файла будут использовать Wireguard1 вместо исходного интерфейса
  - file: common_dns_groups.yaml
    interfaceId: Wireguard1
```

Форма `file:` + `interfaceId` удобна когда один и тот же файл с общими группами используется на роутерах с разными WireGuard-интерфейсами (например, `Wireguard0` на одном роутере, `Wireguard1` на другом) — имена интерфейсов задаются один раз в конфиге каждого роутера, без дублирования файла с группами.

Относительные локальные пути `domain-file` и `domain-url` внутри импортированного файла групп отсчитываются от каталога этого файла.

**Примеры использования:**
- Направить трафик социальных сетей через VPN (Wireguard0)
- Направить потоковые сервисы через другое соединение
- Разделить трафик по доменам для балансировки нагрузки или конфиденциальности
- Использовать поддерживаемые сообществом списки доменов из URL

#### `delete-dns-routing`

*Псевдонимы: `deletednsrouting`, `ddnsr`, `deletednsroutes`, `delete-dns-routes`*

Удаляет все группы DNS-маршрутизации на интерфейсах из конфигурационного файла, включая группы, имена которых в нём не указаны. Для выбора одного интерфейса используйте `--interface-id`.

```shell
# Удалить правила DNS-маршрутизации с подтверждением
./gokeenapi delete-dns-routing --config my_config.yaml

# Удалить правила DNS-маршрутизации без подтверждения
./gokeenapi delete-dns-routing --config my_config.yaml --force
```

Команда выполнит:
1. Определит все dns-proxy маршруты и object-groups на выбранных интерфейсах
2. Отобразит правила для удаления
3. Запросит подтверждение (если не указан флаг `--force`)
4. Удалит сначала dns-proxy маршруты, затем object-groups

#### `delete-all-dns-routing`

*Псевдонимы: `deletealldnsrouting`, `dadnsr`*

Удаляет **все** группы DNS-маршрутизации с роутера одной операцией, независимо от того, что описано в конфигурационном файле. Полезно для полного сброса DNS-маршрутизации перед повторным применением новой или изменённой конфигурации.

```shell
# Удалить все группы DNS-маршрутизации (с подтверждением)
./gokeenapi delete-all-dns-routing --config my_config.yaml

# Удалить все группы DNS-маршрутизации без подтверждения
./gokeenapi delete-all-dns-routing --config my_config.yaml --force
```

> **Внимание:** Эта команда удаляет ВСЕ группы DNS-маршрутизации и dns-proxy маршруты на роутере, включая группы без маршрутов и маршруты без групп. Используйте перед повторным применением после переименования или реорганизации групп.

**Требования:** Прошивка Keenetic версии 5.0.1 или выше

#### `add-awg`

*Псевдонимы: `addawg`, `aawg`*

Добавляет новое WireGuard соединение из `.conf` файла.

```shell
./gokeenapi add-awg --config my_config.yaml --conf-file <путь-к-conf> --name МойСуперИнтерфейс
```

#### `update-awg`

*Псевдонимы: `updateawg`, `uawg`*

Обновляет существующее WireGuard соединение из `.conf` файла. Утилита определяет обычный WireGuard, AWG 1.0/2.0 и AWG 3.1 по полям файла. Для AWG 3.1 поддерживаются `HeaderProtectionKey`, `ContentPaddingAddition`, `RekeyAfterTime`, `RekeyTimeout`, `RejectAfterTime`, `KeepaliveTimeout`, `MaxHandshakeAttempts`, `RandomTrailers` и `DisableCookies`; `H1`–`H4` также могут быть диапазонами. Требуется KeeneticOS 5.02.A.11 или новее. Новые параметры ASC отправляются единым полным блоком RCI. `add-awg` принимает те же файлы. Поддержка AWG 3.1 остаётся экспериментальной до проверки на реальном роутере.

Встроенная настройка peer в Keenetic принимает только одно значение keepalive. Для `PersistentKeepalive = 25-35` утилита использует нижнюю границу (`25`) при импорте и обновлении. `--dry-run` показывает поля 3.1, но вместо `HeaderProtectionKey` выводит только короткий отпечаток SHA-256.

```shell
./gokeenapi update-awg --config my_config.yaml --conf-file <путь-к-conf> --interface-id <interface-id>

# Предпросмотр изменений без применения
./gokeenapi update-awg --config my_config.yaml --conf-file <путь-к-conf> --interface-id <interface-id> --dry-run
```

> **Совет:** Чтобы найти ID интерфейсов, выполните команду `show-interfaces`. Используйте `--dry-run` для просмотра unified diff перед применением изменений.

#### `delete-known-hosts`

*Псевдонимы: `deleteknownhosts`, `dkh`*

Удаляет известные хосты по имени или MAC используя regex паттерн.

```shell
# Удалить хосты по паттерну имени
./gokeenapi delete-known-hosts --config my_config.yaml --name-pattern "паттерн"

# Удалить хосты по паттерну MAC
./gokeenapi delete-known-hosts --config my_config.yaml --mac-pattern "паттерн"

# Удалить хосты без подтверждения
./gokeenapi delete-known-hosts --config my_config.yaml --name-pattern "паттерн" --force
```

#### `scheduler`

*Псевдонимы: `schedule`, `sched`*

Запускает автоматизированные задачи по расписанию или через заданные интервалы. Полную документацию см. в [документации Scheduler](SCHEDULER_RU.md).

```shell
./gokeenapi scheduler --config scheduler.yaml
```

#### `exec`

*Псевдонимы: `e`*

Выполняет пользовательские команды CLI Keenetic (Netcraze) напрямую на вашем роутере.

```shell
# Показать информацию о системе
./gokeenapi exec --config my_config.yaml show version

# Отобразить статистику интерфейсов
./gokeenapi exec --config my_config.yaml show interface

# Показать таблицу маршрутизации
./gokeenapi exec --config my_config.yaml show ip route
```

---

### 🤝 Участие в разработке

Мы приветствуем вклад в проект! Если у вас есть идеи, предложения или сообщения об ошибках, пожалуйста, создайте issue или pull request.

---

### 📄 Лицензия

Этот проект лицензирован под лицензией MIT. Подробности смотрите в файле [LICENSE](LICENSE).
