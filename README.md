# Инструкция по клонированию и запуску проекта

Этот документ описывает шаги для клонирования репозитория, настройки окружения и запуска проекта с использованием Docker Compose.

---

## 1. Установите необходимые инструменты

Перед началом убедитесь, что на вашем компьютере установлены следующие инструменты:

- Git: для клонирования репозитория.  
  [Скачать Git](https://git-scm.com/)
- Docker: для контейнеризации приложений.  
  [Скачать Docker](https://www.docker.com/products/docker-desktop)
- Docker Compose: для управления многоконтейнерными приложениями.  
  Docker Compose встроен в Docker Desktop.

---

## 2. Клонируйте репозиторий

Выполните команду для клонирования по ssh:

```bash
git clone git@github.com:NazarovDA/test_bank.git
```

или

Выполните команду для клонирования по http:

```bash
git clone https://github.com/NazarovDA/test_bank.git
```

## 3. Запуск системы

Перейдите в папку проекта

```bash
cd test_bank
```

Запустите сборку контейнера

```bash
docker-compose up --build
```

## 4. Инструкция по работе с приложением

После того, как api-service будет поднят, по адресу localhost:8080/docs будет доступна страница с описанием эндпоинтов, сгенерированная с помощью https://editor-next.swagger.io


### 5. Описание сервиса и архитектурных решений.

1. Сервис представляет собой REST API, который предоставляет функции для работы с банковскими
счетами.  
Сервис состоит из трех микросервисов:  
`api-service`  
`transaction-worker`  
`transaction-worker`

2. Общение между микросервисами происходит через rabbitmq, что позволяет обеспечить функционал очереди, будет гарантировать, что сообщения не пропадут.

3. В качестве бд выступает postgresql, в которой созданы 2 таблицы - для транзакций и логов. Постольку поскольку проект тестовый, отдельная база данных для логов избыточна, но в более серьезных проектах следует использовать отдельную базу данных, хорошо подойдет ClickHouse.

4. Все запросы обрабатываются в следующем порядке:  
- `api-service` получает запрос, присваивает ему uuid и передает в две очереди в rabbitmq: в логгер и воркер соотвественно.  

- `logger` получает информацию о поступившем запросе и записывает ее в бд со статусом `Pending`, который так же был передан из `api-service`.  

- `worker` получает запрос из своей очереди rabbitmq и производит его дальнейшую обработку. В зависимости от результатов, будет отправлен в `logger` результат операции. В статус будет вписано либо `successful`, если транзакция прошла успешно, либо будет выписана ошибка, которая помешала проведению транзакции, например, в случае недостатка средств будет статус `not enough money`.

5. Каждый сервис регулярно проверяет другие сервисы, от которых он зависит. В случае ошибки, сервисы будут пытаться восстановиться, благодаря директиве docker-compose `restart: always`  
Если сервис был выключен в ручную, то все зависимые сервисы упадут и будут пытаться запуститься вновь. Если поменять условие в `depends_on` на `service_healthy` вместо `service_started`, то это приведет к тому, что в случае, если сервер не успел подняться за заданное время, упадет весь контейнер.

6. Доставку гарантирует `rabbitmq`. В случае, если сообщение им было получено, он будет гарантировать то, что передаст другому микросервису благодаря `DeliveryMode: amqp.Persistent` и настройке очередей.

7. Т.к. история запросов хранится в бд, то к ней всегда будет доступ и она будет сохранена.