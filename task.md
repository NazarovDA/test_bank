
Перевод денег между клиентами банка со счёта на счёт. Нужно реализовать систему транзакций. 
- есть два клиента банка
- первый клиент делает отправку денег второму клиенту

Как происходит транзакция: 
Идет запрос на сервер от клиента, запрос попадает в очередь, которую должны обрабатывать воркеры и выполнять работу по переводу.

Важно: 
- Избежать потери данных при параллельном выполнении запросов
- Код должен соблюдать принципы SOLID,KISS,DRY 
- предусмотреть различные ситуации остановки\перезагрузки  сервера: плановая остановка и перезапуск, падение
- история запросов не должна пропасть при перезапуске сервера
- Для реализации очереди можно использовать RabbitMQ

Что нужно реализовать : 
- БД на postgresql, где будет схема с клиентами и их балансами
- сервер, который проверяет все условия (например: хватает ли денег для совершения операции) и делает изменение баланса (на + или -)
- Вся инфраструктура должна легко подниматься через Docker, написать Dockerfile, docker-compose