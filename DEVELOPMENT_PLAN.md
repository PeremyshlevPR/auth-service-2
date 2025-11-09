# Development Plan - Auth Service

## Обзор проекта

Auth Service - микросервис для аутентификации и авторизации пользователей. Сервис обеспечивает регистрацию, вход, управление сессиями и токенами, с возможностью расширения для интеграции с внешними OAuth2 провайдерами.

## Технологический стек

- **Язык**: Go 1.21+
- **HTTP Framework**: Gin или Echo (на выбор)
- **База данных**: PostgreSQL 15+
- **Кэш/Сессии**: Redis 7+
- **JWT**: github.com/golang-jwt/jwt/v5
- **Хеширование паролей**: golang.org/x/crypto/bcrypt
- **Конфигурация**: environment variables + Viper (опционально)
- **Миграции БД**: golang-migrate/migrate
- **Валидация**: go-playground/validator
- **Логирование**: logrus или zap
- **Тестирование**: testify
- **Документация API**: Swagger/OpenAPI

## Архитектура

### Компоненты

1. **Auth Service** - основной сервис аутентификации
2. **PostgreSQL** - хранение пользователей, refresh tokens
3. **Redis** - черные списки токенов, активные сессии, rate limiting

### Поток авторизации

1. Пользователь регистрируется/входит → получает access token + refresh token
2. Access token передается в заголовке `Authorization: Bearer <token>`
3. Refresh token хранится в httpOnly cookie или возвращается в ответе
4. При истечении access token → обновление через refresh token
5. При logout → refresh token добавляется в blacklist (Redis)

## Функциональность MVP (Минимальная версия)

### 1. Регистрация пользователя
- **Endpoint**: `POST /api/v1/auth/register`
- **Функционал**:
  - Валидация email и пароля
  - Проверка уникальности email
  - Хеширование пароля (bcrypt)
  - Создание пользователя в БД
  - Генерация пары токенов (access + refresh)
  - Возврат access token в ответе
  - Сохранение refresh token в БД и установка в httpOnly cookie

### 2. Вход пользователя
- **Endpoint**: `POST /api/v1/auth/login`
- **Функционал**:
  - Валидация email и пароля
  - Проверка существования пользователя
  - Проверка пароля (bcrypt)
  - Генерация пары токенов
  - Возврат access token в ответе
  - Установка refresh token в httpOnly cookie
  - Сохранение refresh token в БД

### 3. Обновление токенов
- **Endpoint**: `POST /api/v1/auth/refresh`
- **Функционал**:
  - Получение refresh token из cookie
  - Валидация refresh token
  - Проверка отсутствия в blacklist (Redis)
  - Проверка существования в БД
  - Генерация новой пары токенов
  - Инвалидация старого refresh token (blacklist + удаление из БД)
  - Возврат нового access token
  - Установка нового refresh token в cookie

### 4. Выход пользователя
- **Endpoint**: `POST /api/v1/auth/logout`
- **Функционал**:
  - Получение refresh token из cookie
  - Добавление refresh token в blacklist (Redis)
  - Удаление refresh token из БД
  - Очистка cookie
  - Добавление access token в blacklist (опционально, для текущей сессии)

### 5. Валидация токена (Middleware)
- **Middleware**: `AuthMiddleware`
- **Функционал**:
  - Извлечение access token из заголовка Authorization
  - Валидация JWT токена
  - Проверка наличия в blacklist (Redis)
  - Добавление user_id и claims в контекст запроса
  - Проверка срока действия токена

### 6. Получение профиля текущего пользователя
- **Endpoint**: `GET /api/v1/auth/me`
- **Функционал**:
  - Требует авторизации (AuthMiddleware)
  - Возврат информации о текущем пользователе (email, user_id, created_at)

## Схема базы данных

### Таблица: users
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE,
    is_email_verified BOOLEAN DEFAULT FALSE
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_created_at ON users(created_at);
```

### Таблица: refresh_tokens
```sql
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    device_info VARCHAR(255),
    ip_address VARCHAR(45)
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
```

### Таблица: oauth_providers (для будущей интеграции)
```sql
CREATE TABLE oauth_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL, -- 'google', 'apple', 'facebook'
    provider_user_id VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(provider, provider_user_id)
);

CREATE INDEX idx_oauth_providers_user_id ON oauth_providers(user_id);
CREATE INDEX idx_oauth_providers_provider ON oauth_providers(provider);
```

## Структура проекта

```
auth-service-2/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── domain/
│   │   ├── user.go
│   │   ├── token.go
│   │   └── oauth_provider.go
│   ├── repository/
│   │   ├── user_repository.go
│   │   ├── token_repository.go
│   │   └── interfaces.go
│   ├── service/
│   │   ├── auth_service.go
│   │   ├── token_service.go
│   │   └── interfaces.go
│   ├── handler/
│   │   ├── auth_handler.go
│   │   └── middleware.go
│   ├── dto/
│   │   └── requests.go
│   └── utils/
│       ├── jwt.go
│       ├── password.go
│       └── validator.go
├── migrations/
│   ├── 000001_init_users.up.sql
│   ├── 000001_init_users.down.sql
│   ├── 000002_init_refresh_tokens.up.sql
│   ├── 000002_init_refresh_tokens.down.sql
│   ├── 000003_init_oauth_providers.up.sql
│   └── 000003_init_oauth_providers.down.sql
├── pkg/
│   └── database/
│       ├── postgres.go
│       └── redis.go
├── docker-compose.yml
├── Dockerfile
├── .env.example
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Конфигурация

### Environment Variables

```env
# Server
SERVER_PORT=8080
SERVER_HOST=0.0.0.0
SERVER_READ_TIMEOUT=15s
SERVER_WRITE_TIMEOUT=15s

# Database
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=auth_service
POSTGRES_PASSWORD=password
POSTGRES_DB=auth_service_db
POSTGRES_SSLMODE=disable

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# JWT
JWT_SECRET=your-secret-key-change-in-production
JWT_ACCESS_TOKEN_EXPIRY=15m
JWT_REFRESH_TOKEN_EXPIRY=7d

# Security
BCRYPT_COST=12
RATE_LIMIT_REQUESTS=10
RATE_LIMIT_WINDOW=1m

# CORS
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:3001
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Content-Type,Authorization
```

## Безопасность

### Реализованные меры

1. **Хеширование паролей**: bcrypt с cost=12
2. **JWT токены**: подпись с секретным ключом
3. **HTTPOnly cookies**: для refresh tokens (защита от XSS)
4. **HTTPS only**: все запросы через HTTPS
5. **Rate limiting**: ограничение количества запросов на эндпоинты входа/регистрации
6. **Blacklist токенов**: Redis для инвалидированных токенов
7. **Валидация входных данных**: проверка email, пароля (минимальная длина, сложность)
8. **SQL injection protection**: параметризованные запросы
9. **CORS**: настройка разрешенных источников

### Парольные требования

- Минимальная длина: 8 символов
- Рекомендуется: заглавные, строчные буквы, цифры, специальные символы

## API Endpoints

### Public endpoints

- `POST /api/v1/auth/register` - Регистрация
- `POST /api/v1/auth/login` - Вход
- `POST /api/v1/auth/refresh` - Обновление токенов
- `POST /api/v1/auth/logout` - Выход

### Protected endpoints

- `GET /api/v1/auth/me` - Получение профиля текущего пользователя

## Будущие улучшения (Roadmap)

### Фаза 2: Управление профилем и безопасность

- [ ] Смена пароля (требует текущий пароль)
- [ ] Восстановление пароля (email с токеном сброса)
- [ ] Подтверждение email (верификация через email)
- [ ] Управление активными сессиями (просмотр и отзыв устройств)
- [ ] История входов (логирование всех попыток входа)

### Фаза 3: OAuth2 интеграция

- [ ] Интеграция с Google OAuth2
- [ ] Интеграция с Apple Sign In
- [ ] Интеграция с Facebook OAuth2
- [ ] Связывание аккаунтов (привязка OAuth к существующему аккаунту)
- [ ] Отвязка OAuth провайдеров
- [ ] Endpoint для инициации OAuth flow: `GET /api/v1/auth/oauth/:provider`

### Фаза 4: Двухфакторная аутентификация (2FA)

- [ ] Настройка 2FA (TOTP через приложения типа Google Authenticator)
- [ ] Вход с 2FA кодом
- [ ] Резервные коды для восстановления
- [ ] Отключение 2FA

### Фаза 5: Роли и права доступа

- [ ] Система ролей (user, premium, admin)
- [ ] RBAC (Role-Based Access Control)
- [ ] Permissions в JWT claims
- [ ] Middleware для проверки прав доступа

### Фаза 6: Мониторинг и аналитика

- [ ] Метрики (Prometheus)
- [ ] Логирование структурированное (JSON)
- [ ] Трейсинг (Jaeger/OpenTelemetry)
- [ ] Health checks (`/health`, `/ready`)
- [ ] Audit log (логирование всех действий пользователей)

### Фаза 7: Расширенные функции

- [ ] OAuth2 Provider (возможность быть провайдером для других сервисов)
- [ ] API keys для сервис-сервисной авторизации
- [ ] Webhook уведомления о событиях (регистрация, вход, смена пароля)
- [ ] Геолокация и ограничение доступа по регионам
- [ ] Device fingerprinting для дополнительной безопасности

## План разработки MVP

### Этап 1: Настройка проекта
- [x] Инициализация Go модуля
- [ ] Настройка структуры проекта
- [ ] Настройка Docker Compose (PostgreSQL + Redis)
- [ ] Настройка конфигурации
- [ ] Создание миграций БД

### Этап 2: Базовая инфраструктура
- [ ] Подключение к PostgreSQL
- [ ] Подключение к Redis
- [ ] Реализация репозиториев (User, Token)
- [ ] Реализация утилит (JWT, Password hashing, Validator)

### Этап 3: Сервисный слой
- [ ] Реализация AuthService
- [ ] Реализация TokenService
- [ ] Бизнес-логика регистрации
- [ ] Бизнес-логика входа
- [ ] Бизнес-логика обновления токенов
- [ ] Бизнес-логика выхода

### Этап 4: HTTP handlers
- [ ] Реализация AuthHandler
- [ ] Реализация middleware (Auth, CORS, Rate limiting, Logging)
- [ ] Регистрация роутов
- [ ] Обработка ошибок

### Этап 5: Тестирование
- [ ] Unit тесты для сервисов
- [ ] Unit тесты для репозиториев
- [ ] Integration тесты для API endpoints
- [ ] Тесты безопасности

### Этап 6: Документация
- [ ] API документация (Swagger/OpenAPI)
- [ ] README с инструкциями по запуску
- [ ] Примеры использования API

## Метрики успеха MVP

- ✅ Регистрация пользователя работает
- ✅ Вход пользователя работает
- ✅ Обновление токенов работает
- ✅ Выход пользователя работает
- ✅ Middleware корректно валидирует токены
- ✅ Все endpoints покрыты тестами
- ✅ Документация API готова
- ✅ Сервис готов к деплою

## Примечания

- Все пароли должны хешироваться перед сохранением в БД
- Refresh tokens должны храниться с хешем (не в открытом виде)
- Access tokens должны быть короткоживущими (15 минут)
- Refresh tokens должны быть долгоживущими (7 дней)
- При выходе refresh token должен быть добавлен в blacklist
- Все токены должны проверяться на наличие в blacklist перед использованием
- Сервис должен быть stateless (кроме refresh tokens в БД)
- Все ошибки должны быть логированы, но не раскрывать внутреннюю информацию

