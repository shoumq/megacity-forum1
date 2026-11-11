# MegaCity

Прототип сообщества MegaCity: интерфейс на React/Vite, API на Go и демонстрационный магазин внутриигровых предметов.

## Структура

- `frontend/` — сайт и адаптивная страница магазина.
- `backend/` — Go API регистрации, входа, профиля, каталога и покупок.
- `docker-compose.yaml` — запуск сайта и API как двух сервисов.

## Запуск через Docker Compose

Нужен установленный Docker с Compose plugin. Из корня проекта:

```sh
MEGACITY_SECRET="длинный-случайный-секрет" docker compose up --build
```

Откройте <http://localhost:5173>. Для PowerShell задайте переменную перед запуском:

```powershell
$env:MEGACITY_SECRET = "длинный-случайный-секрет"
docker compose up --build
```

Данные аккаунтов и балансы сохраняются в Docker volume `megacity-data`.

## Локальный запуск без Docker

В первом терминале:

```sh
cd backend
go run .
```

Во втором:

```sh
cd frontend
npm install
npm run dev -- --host 0.0.0.0
```

Vite перенаправляет `/api` на `http://localhost:8080`.

## API

- `GET /api/health` — состояние API.
- `POST /api/auth/register` — `{ "nickname", "email", "password" }`.
- `POST /api/auth/login` — `{ "login", "password" }`; `login` принимает ник или почту.
- `GET /api/me` — профиль и баланс, заголовок `Authorization: Bearer <token>`.
- `GET /api/shop/products` — каталог магазина.
- `POST /api/shop/buy` — `{ "productId" }`, авторизованный запрос.

Новый аккаунт получает тестовый баланс 1 000 MC. Покупки и баланс хранятся на Go-сервере. Сейчас магазин не подключён к плагину/моду Minecraft, игровой экономике или платёжной системе; товары и MC — демонстрационные данные прототипа.

Для развёртывания замените `MEGACITY_SECRET` на длинный уникальный секрет. Сейчас хранилище — локальный JSON-файл, рассчитанный на один экземпляр API; перед публичным запуском следует перенести аккаунты и транзакции в базу данных и подключить авторитетную экономику игрового сервера.
