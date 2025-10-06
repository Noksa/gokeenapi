<div align="center">

# 🚀 gokeenapi

**Автоматизируйте управление роутером Keenetic с легкостью**

<p align="center">
  <video src="https://github.com/user-attachments/assets/404e89cc-4675-42c4-ae93-4a0955b06348" width="100%"></video>
</p>

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Docker Pulls](https://img.shields.io/docker/pulls/noksa/gokeenapi)](https://hub.docker.com/r/noksa/gokeenapi)
[![GitHub release](https://img.shields.io/github/release/Noksa/gokeenapi.svg)](https://github.com/Noksa/gokeenapi/releases)

*Устали кликать по веб-интерфейсу Keenetic? Автоматизируйте управление роутером с помощью простых CLI команд.*

[🚀 Быстрый старт](#-быстрый-старт) • [📖 Документация](#-команды) • [🎨 GUI версия](https://github.com/Noksa/gokeenapiui) • [🤝 Участие в разработке](#-участие-в-разработке)

</div>

---

## ✨ Почему стоит выбрать gokeenapi?

<table>
<tr>
<td width="50%">

### 💻 **Автоматизируйте всё**
Управляйте маршрутами, DNS записями, WireGuard соединениями и известными хостами простыми командами

### ⚙️ **Без настройки роутера**
Не нужна сложная конфигурация роутера - просто укажите адрес

</td>
<td width="50%">

### 🌐 **Работает везде**
Доступ по LAN или через интернет via KeenDNS - на ваш выбор

### 🎯 **Точное управление**
Удаляйте статические маршруты для конкретных интерфейсов, не затрагивая другие

</td>
</tr>
</table>

---

## 🎨 Предпочитаете GUI?

Не любите командную строку? У нас есть решение! Попробуйте нашу удобную GUI версию:

<div align="center">

### [🎨 **Доступна GUI версия** 🚀](https://github.com/Noksa/gokeenapiui)

[![GUI Version](https://img.shields.io/badge/🎨_Попробовать_GUI-Нажмите_здесь-brightgreen?style=for-the-badge&logo=github)](https://github.com/Noksa/gokeenapiui)

</div>

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

`gokeenapi` настраивается с помощью `yaml` файла. Пример можно найти [здесь](https://github.com/Noksa/gokeenapi/blob/main/config_example.yaml).

Чтобы использовать ваш конфигурационный файл, передайте флаг `--config <путь>` с вашей командой.

### Переменные окружения

Для безопасности вы можете хранить чувствительные данные в переменных окружения вместо конфигурационного файла:

- `GOKEENAPI_KEENETIC_LOGIN` - Логин администратора роутера
- `GOKEENAPI_KEENETIC_PASSWORD` - Пароль администратора роутера

---

## 🔧 Поддерживаемые роутеры

`gokeenapi` протестирован со следующими моделями роутеров Keenetic:

- **Keenetic Start**
- **Keenetic Viva** 
- **Keenetic Giga**

Поскольку утилита работает с Keenetic Start (самая доступная модель в линейке), она должна быть совместима со всеми моделями роутеров Keenetic.

---

## 🎬 Видео демонстрации

Посмотрите эти видео демонстрации, чтобы увидеть `gokeenapi` в действии:

*   [Управление маршрутами](https://www.youtube.com/watch?v=lKX74btFypY)

---

### 📚 Команды

Вот некоторые вещи, которые вы можете делать с `gokeenapi`. Для полного списка команд и опций используйте флаг `--help`.

```shell
./gokeenapi --help
```

#### `show-interfaces`

*Псевдонимы: `showinterfaces`, `showifaces`, `si`*

Отображает все доступные интерфейсы на вашем роутере Keenetic.

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

---

### 🤝 Участие в разработке

Мы приветствуем вклад в проект! Если у вас есть идеи, предложения или сообщения об ошибках, пожалуйста, создайте issue или pull request.

---

### 📄 Лицензия

Этот проект лицензирован под лицензией MIT. Подробности смотрите в файле [LICENSE](LICENSE).
