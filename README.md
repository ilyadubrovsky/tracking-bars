# Tracking BARS – БАРС  «МЭИ» в телеграм.

[![Telegram bot](https://img.shields.io/badge/telegram-bot-0088cc.svg)](https://t.me/trackingbarsbot)

Бот позволяет взаимодействовать с БАРС в телеграм. Вы можете смотреть оценки в удобной форме и получать уведомления об их изменениях.

## Решения
+ Фреймворк [Telebot](https://github.com/tucnak/telebot) для для взаимодействия Telegram API;
+ СУБД PostgreSQL для хранения данных (на SQL запросах): [pgx](https://github.com/jackc/pgx);
+ Конфигурация приложения: [cleanenv](https://github.com/ilyakaznacheev/cleanenv) и [godotenv](https://github.com/joho/godotenv);
+ Многоуровневое логирование: [logrus](https://github.com/sirupsen/logrus);
+ Docker и надстройка docker-compose для развёртывания;
+ Многопоточный парсинг данных пользователей;
+ Упрощённая чистая архитектура;
+ AES/CFB шифрование паролей.

#### Структура проекта:
    .
    ├── cmd/main                    # точка входа в приложение
    ├── configs                     # файлы конфигурации
    ├── internal                    # внутренняя логика приложения
    │   ├── app                     # рабочая область приложения
    │   ├── config                  # структура конфигурации
    │   ├── entity                  # сущности приложения
    │       ├── change
    │       ├── user
    │   ├── service                 # бизнес-логика приложения (пока только БАРС)
    │       ├── bars                # барс сервис (парсинг)
    │       ├── telegram            # телеграм БОТ сервис 
    │   ├── storage                 # реализация базы данных
    ├── pkg                         # вспомогательные экспортируемые пакеты
    │   ├── client                  # клиенты
    │       ├── bars
    │       ├── postgresql
    │   ├── logging                 # логгирование приложения
    │   ├── utils                   # дополнительные утилиты
    │       ├── aes
    └── migrations                  # миграции базы данных

## Сборка проекта
1. Клонируйте этот репозиторий: `git clone https://github.com/ilyadubrovsky/tracking-bars`;
2. Создайте `.env` файл и укажите переменные среды, перечисленные ниже;
3. Откройте корневую директорию проекта в командной строке;
4. Выполните сборку проекта: `make build`;
5. Выполните запуск проекта: `make run`;
6. При первом запуске примените миграции к базе данных: `make migrate-up`.

| Переменная       | Описание                                                    |
|------------------|-------------------------------------------------------------|
| `TELEGRAM_TOKEN` | Токен телеграм бота из [@BotFather](https://t.me/BotFather) |
| `ADMIN_ID`       | ChatID администратора                                       |
| `ENCRYPTION_KEY` | 32-битный ключ шифрования паролей                           |
| `PG_HOST`        | Хост, по умолчанию указывать `db`                           |
| `PG_PORT`        | Порт, по умолчанию указывать `5432`                         |
| `PG_USERNAME`    | Имя пользователя, по умолчанию указывать `postgres`         |
| `PG_PASSWORD`    | Пароль, по умолчанию указывать `12345678`                   |
| `PG_DATABASE`    | База данных, по умолчанию указывать `trackingbars`          |
