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

Использование Docker - рекомендуемый способ запуска `gokeenapi`.

```bash
# Скачиваем Docker образ
export GOKEENAPI_IMAGE="noksa/gokeenapi:stable"
docker pull "${GOKEENAPI_IMAGE}"

# Запускаем команду
docker run --rm -ti -v "$(pwd)/config_example.yaml":/gokeenapi/config.yaml \
  "${GOKEENAPI_IMAGE}" show-interfaces --config /gokeenapi/config.yaml
```

### 📦 Последний релиз

Скачайте последний релиз для вашей платформы:

<div align="center">

[![Download Latest](https://img.shields.io/badge/📦_Скачать-Последний_релиз-green?style=for-the-badge)](https://github.com/Noksa/gokeenapi/releases)

</div>

---

## ⚙️ Конфигурация

`gokeenapi` настраивается с помощью `yaml` файла. Пример можно найти [здесь](https://github.com/Noksa/gokeenapi/blob/main/config_example.yaml). Полное описание каждого поля смотрите в [Справочнике конфигурации](docs/config-reference-ru.md).

Чтобы использовать ваш конфигурационный файл, передайте флаг `--config <путь>` с вашей командой.

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
| `GOKEENAPI_CONFIG` | Путь к файлу конфигурации (альтернатива флагу `--config`) |
| `GOKEENAPI_KEENETIC_LOGIN` | Логин администратора роутера |
| `GOKEENAPI_KEENETIC_PASSWORD` | Пароль администратора роутера |
| `GOKEENAPI_INSIDE_DOCKER` | Если задана, использует `/etc/gokeenapi` в качестве директории данных |

`GOKEENAPI_KEENETIC_LOGIN` и `GOKEENAPI_KEENETIC_PASSWORD` позволяют не хранить чувствительные данные в конфигурационном файле. `GOKEENAPI_INSIDE_DOCKER` задаётся автоматически в официальном Docker-образе.

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

> **Примечание**: Используйте `tls_skip_verify` только в доверенных локальных сетях. Отключение проверки сертификата делает соединение уязвимым к атакам типа «человек посередине».

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
- Создает группы доменов (object-groups), содержащие указанные домены и IP-адреса
- Связывает каждую группу с сетевым интерфейсом через dns-proxy маршруты
- Трафик для доменов в группе автоматически направляется через указанный интерфейс

**Источники доменов:**
- Локальные .txt файлы с одним доменом на строку (поддерживаются комментарии с #)
- Удаленные URL с списками доменов
- YAML файлы, содержащие списки путей к domain-file или domain-url (для организации)

**Раскрытие YAML:** Утилита автоматически определяет `.yaml`/`.yml` файлы в массивах `domain-file` и `domain-url` и раскрывает их в содержащиеся в них пути к доменам (аналогично раскрытию bat-file/bat-url).

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

**Примеры использования:**
- Направить трафик социальных сетей через VPN (Wireguard0)
- Направить потоковые сервисы через другое соединение
- Разделить трафик по доменам для балансировки нагрузки или конфиденциальности
- Использовать поддерживаемые сообществом списки доменов из URL

#### `delete-dns-routing`

*Псевдонимы: `deletednsrouting`, `ddnsr`, `deletednsroutes`, `delete-dns-routes`*

Удаляет правила DNS-маршрутизации, соответствующие вашему конфигурационному файлу.

```shell
# Удалить правила DNS-маршрутизации с подтверждением
./gokeenapi delete-dns-routing --config my_config.yaml

# Удалить правила DNS-маршрутизации без подтверждения
./gokeenapi delete-dns-routing --config my_config.yaml --force
```

Команда выполнит:
1. Определит dns-proxy маршруты и object-groups, соответствующие вашей конфигурации
2. Отобразит правила для удаления
3. Запросит подтверждение (если не указан флаг `--force`)
4. Удалит сначала dns-proxy маршруты, затем object-groups

#### `add-awg`

*Псевдонимы: `addawg`, `aawg`*

Добавляет новое WireGuard соединение из `.conf` файла.

```shell
./gokeenapi add-awg --config my_config.yaml --conf-file <путь-к-conf> --name МойСуперИнтерфейс
```

#### `update-awg`

*Псевдонимы: `updateawg`, `uawg`*

Обновляет существующее WireGuard соединение из `.conf` файла.

```shell
./gokeenapi update-awg --config my_config.yaml --conf-file <путь-к-conf> --interface-id <interface-id>
```

> **Совет:** Чтобы найти ID интерфейсов, выполните команду `show-interfaces`.

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
