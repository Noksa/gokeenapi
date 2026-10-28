# Локальный стенд exporter

Стенд запускает exporter из текущего checkout вместе с Prometheus и Grafana. Нужны Docker Engine и Docker Compose v2.24+. Отдельно устанавливать Go не требуется: Compose собирает exporter через Dockerfile репозитория.

Из корня репозитория:

```bash
make exporter-local-up EXPORTER_CONFIG=/path/to/exporter.yaml
make exporter-local-status
make exporter-local-logs
make exporter-local-down
```

`EXPORTER_CONFIG` может быть абсолютным или относительным путём от корня репозитория. Каталог с этим файлом монтируется только для чтения в `/etc/gokeenapi`; пути к роутерам в YAML exporter должны быть относительными и оставаться внутри этого каталога (подкаталоги поддерживаются). Контейнер работает с UID/GID пользователя хоста: приватные файлы `0600` доступны без изменения прав. Exporter загружает только настройки подключения, поэтому файлы маршрутов и доменных списков не нужны.

Например, положите оба файла в приватный каталог:

```yaml
# exporter.yaml
listen: 127.0.0.1:9479
routers:
  - name: home
    config: router.yaml
```

```yaml
# router.yaml
keenetic:
  url: https://router.example.net
  login: monitor
  password: replace-me
```

Стенд передаёт `--listen 0.0.0.0:9479` внутри контейнера exporter; исходный YAML не изменяется. Все порты хоста публикуются на `127.0.0.1`:

| Сервис | URL |
|---|---|
| Grafana | <http://127.0.0.1:3000/d/gokeenapi-keenetic> |
| Prometheus | <http://127.0.0.1:9090> |
| Exporter | <http://127.0.0.1:9479/metrics> |

Grafana автоматически загружает datasource и дашборд, который становится домашней страницей. Анонимный доступ имеет роль Viewer; входа и редактирования дашбордов через UI в этом локальном стенде нет. Дашборд в `.runtime/<project>/dashboards` генерируется из единственного исходника `grafana/keenetic.json` с подстановкой UID datasource для provisioning. После изменений дашборда `make exporter-local-up` генерирует его заново.

Prometheus собирает метрики раз в 15 секунд и хранит историю семь дней. Интервалы опроса exporter сохраняются и не зависят от scrape. Перед анализом значений проверяйте успешность и свежесть коллекторов: доступность endpoint exporter не означает, что доступны все роутеры. Для графиков скорости нужно несколько опросов, обычно 2–5 минут.

Каждый явный `up` пересобирает и пересоздаёт сервисы, применяя изменения YAML exporter/роутеров и provisioning-файлов. История остаётся в именованных volumes. Остановка сохраняет именованные volumes Prometheus/Grafana. Для `down`, `status` и `logs` не нужен `EXPORTER_CONFIG`. Для отдельного стенда задайте одинаковый `EXPORTER_PROJECT` во всех командах. Порты можно изменить:

```bash
make exporter-local-up EXPORTER_CONFIG=/path/to/exporter.yaml \
  EXPORTER_PROJECT=keenetic-lab \
  GRAFANA_PORT=13000 PROMETHEUS_PORT=19090 EXPORTER_PORT=19479
make exporter-local-down EXPORTER_PROJECT=keenetic-lab
```

Если в shell заданы `GOKEENAPI_KEENETIC_LOGIN` и `GOKEENAPI_KEENETIC_PASSWORD`, они передаются только exporter и переопределяют учётные данные всех роутеров. Без них работают данные из YAML. Пароли роутеров не копируются в генерируемые файлы и образы.
