# Yunt - Development Mail Server

## Product Requirements Document (PRD)

**Version:** 1.0  
**Date:** December 2024  
**Author:** Kerem  
**Status:** Draft

---

## 1. Executive Summary

### 1.1 Proje Tanımı

Yunt, geliştiriciler ve test ortamları için tasarlanmış, Go ile yazılmış hafif ve güçlü bir mail sunucusudur. İsim, Göktürk Türkçesinde "at" anlamına gelir - tıpkı atlı ulakların mektupları taşıdığı gibi, Yunt da e-postaları güvenle taşır.

### 1.2 Vizyon

Geliştirme ortamlarında mail testini kolaylaştıran, kurulumu basit, kullanımı sezgisel ve her türlü veritabanı altyapısına entegre edilebilen bir mail çözümü.

### 1.3 Hedef Kitle

- Backend geliştiriciler
- QA mühendisleri
- DevOps ekipleri
- CI/CD pipeline'ları

### 1.4 Temel Özellikler

- SMTP server (mail yakalama + relay)
- IMAP server (mail client desteği)
- Modern Web UI (admin paneli)
- REST API
- Çoklu kullanıcı desteği
- 4 farklı veritabanı desteği (SQLite, PostgreSQL, MySQL, MongoDB)

---

## 2. Problem Tanımı

### 2.1 Mevcut Sorunlar

| Problem | Açıklama |
|---------|----------|
| **Karmaşıklık** | Postfix, Dovecot gibi production çözümleri geliştirme için aşırı kompleks |
| **Test riski** | Gerçek adreslere yanlışlıkla mail gönderimi tehlikesi |
| **Veritabanı bağımlılığı** | Mevcut araçlar genellikle tek veritabanına bağımlı |
| **IMAP eksikliği** | Mailhog, Mailpit gibi araçlar IMAP desteği sunmuyor |
| **İzolasyon** | Ekip çalışması için yetersiz kullanıcı/mailbox izolasyonu |

### 2.2 Mevcut Alternatifler

| Araç | SMTP | IMAP | Web UI | Multi-DB | Multi-User |
|------|------|------|--------|----------|------------|
| Mailhog | ✓ | ✗ | ✓ | ✗ | ✗ |
| Mailpit | ✓ | ✗ | ✓ | ✗ | ✗ |
| Mailtrap | ✓ | ✗ | ✓ | - | ✓ |
| **Yunt** | ✓ | ✓ | ✓ | ✓ | ✓ |

---

## 3. Sistem Mimarisi

### 3.1 Üst Düzey Mimari

```
┌─────────────────────────────────────────────────────────────────┐
│                          CLIENTS                                │
├─────────────────┬─────────────────┬─────────────────────────────┤
│   Mail Client   │   Web Browser   │   Application (SMTP)        │
│  (Thunderbird)  │                 │   (Laravel, Django, etc.)   │
└────────┬────────┴────────┬────────┴─────────────┬───────────────┘
         │ IMAP            │ HTTP                  │ SMTP
         │ :1143           │ :8025                 │ :1025
┌────────▼─────────────────▼──────────────────────▼───────────────┐
│                                                                 │
│                         YUNT SERVER                             │
│                                                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │ IMAP Server  │  │  API Server  │  │    SMTP Server       │   │
│  │              │  │   + Web UI   │  │                      │   │
│  └──────┬───────┘  └──────┬───────┘  └──────────┬───────────┘   │
│         │                 │                      │               │
│         └─────────────────┼──────────────────────┘               │
│                           │                                      │
│                   ┌───────▼───────┐                              │
│                   │  Core Engine  │                              │
│                   │   Services    │                              │
│                   └───────┬───────┘                              │
│                           │                                      │
│                   ┌───────▼───────┐                              │
│                   │  Repository   │                              │
│                   │    Layer      │                              │
│                   └───────┬───────┘                              │
│                           │                                      │
└───────────────────────────┼──────────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
        ▼                   ▼                   ▼
   ┌─────────┐        ┌──────────┐        ┌─────────┐
   │ SQLite  │        │PostgreSQL│        │  MySQL  │
   │         │        │  MySQL   │        │         │
   └─────────┘        └──────────┘        └─────────┘
                            │
                            ▼
                      ┌──────────┐
                      │ MongoDB  │
                      └──────────┘
```

### 3.2 Bileşen Detayları

```
┌─────────────────────────────────────────────────────────────────┐
│                       YUNT COMPONENTS                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    PROTOCOL LAYER                        │    │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐  │    │
│  │  │    SMTP     │  │    IMAP     │  │   HTTP/REST     │  │    │
│  │  │   Server    │  │   Server    │  │     Server      │  │    │
│  │  │  (go-smtp)  │  │  (go-imap)  │  │   (echo/fiber)  │  │    │
│  │  └─────────────┘  └─────────────┘  └─────────────────┘  │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                   │
│  ┌───────────────────────────▼─────────────────────────────┐    │
│  │                    SERVICE LAYER                         │    │
│  │  ┌───────────┐ ┌───────────┐ ┌───────────┐ ┌─────────┐  │    │
│  │  │   User    │ │  Mailbox  │ │  Message  │ │  Relay  │  │    │
│  │  │  Service  │ │  Service  │ │  Service  │ │ Service │  │    │
│  │  └───────────┘ └───────────┘ └───────────┘ └─────────┘  │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                   │
│  ┌───────────────────────────▼─────────────────────────────┐    │
│  │                   REPOSITORY LAYER                       │    │
│  │  ┌─────────────────────────────────────────────────┐    │    │
│  │  │            Repository Interface                  │    │    │
│  │  │  - UserRepository                                │    │    │
│  │  │  - MailboxRepository                             │    │    │
│  │  │  - MessageRepository                             │    │    │
│  │  │  - AttachmentRepository                          │    │    │
│  │  │  - SettingsRepository                            │    │    │
│  │  └─────────────────────────────────────────────────┘    │    │
│  │         │            │            │            │         │    │
│  │    ┌────▼────┐  ┌────▼────┐  ┌────▼────┐  ┌───▼─────┐   │    │
│  │    │ SQLite  │  │ Postgres│  │  MySQL  │  │ MongoDB │   │    │
│  │    │ Adapter │  │ Adapter │  │ Adapter │  │ Adapter │   │    │
│  │    └─────────┘  └─────────┘  └─────────┘  └─────────┘   │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 3.3 Dizin Yapısı

```
yunt/
├── cmd/
│   └── yunt/
│       └── main.go                 # Ana giriş noktası
│
├── internal/
│   ├── config/
│   │   ├── config.go               # Konfigürasyon yapısı
│   │   └── loader.go               # YAML/ENV loader
│   │
│   ├── domain/
│   │   ├── user.go                 # User entity
│   │   ├── mailbox.go              # Mailbox entity
│   │   ├── message.go              # Message entity
│   │   ├── attachment.go           # Attachment entity
│   │   ├── webhook.go              # Webhook entity
│   │   └── errors.go               # Domain errors
│   │
│   ├── repository/
│   │   ├── repository.go           # Interface definitions
│   │   ├── factory.go              # Repository factory
│   │   ├── sqlite/
│   │   │   ├── sqlite.go           # SQLite connection
│   │   │   ├── user.go             # User repository
│   │   │   ├── mailbox.go          # Mailbox repository
│   │   │   ├── message.go          # Message repository
│   │   │   └── migrations.go       # SQLite migrations
│   │   ├── postgres/
│   │   │   ├── postgres.go
│   │   │   ├── user.go
│   │   │   ├── mailbox.go
│   │   │   ├── message.go
│   │   │   └── migrations.go
│   │   ├── mysql/
│   │   │   ├── mysql.go
│   │   │   ├── user.go
│   │   │   ├── mailbox.go
│   │   │   ├── message.go
│   │   │   └── migrations.go
│   │   └── mongodb/
│   │       ├── mongodb.go
│   │       ├── user.go
│   │       ├── mailbox.go
│   │       └── message.go
│   │
│   ├── service/
│   │   ├── user.go                 # User business logic
│   │   ├── mailbox.go              # Mailbox business logic
│   │   ├── message.go              # Message business logic
│   │   ├── auth.go                 # Authentication logic
│   │   ├── relay.go                # SMTP relay logic
│   │   └── webhook.go              # Webhook dispatch
│   │
│   ├── smtp/
│   │   ├── server.go               # SMTP server setup
│   │   ├── backend.go              # SMTP backend handler
│   │   └── session.go              # SMTP session handler
│   │
│   ├── imap/
│   │   ├── server.go               # IMAP server setup
│   │   ├── backend.go              # IMAP backend handler
│   │   ├── user.go                 # IMAP user handler
│   │   └── mailbox.go              # IMAP mailbox handler
│   │
│   ├── api/
│   │   ├── server.go               # HTTP server setup
│   │   ├── router.go               # Route definitions
│   │   ├── middleware/
│   │   │   ├── auth.go             # JWT middleware
│   │   │   ├── cors.go             # CORS middleware
│   │   │   ├── logger.go           # Request logging
│   │   │   └── ratelimit.go        # Rate limiting
│   │   └── handlers/
│   │       ├── auth.go             # Auth endpoints
│   │       ├── users.go            # User endpoints
│   │       ├── mailboxes.go        # Mailbox endpoints
│   │       ├── messages.go         # Message endpoints
│   │       ├── attachments.go      # Attachment endpoints
│   │       ├── search.go           # Search endpoints
│   │       ├── settings.go         # Settings endpoints
│   │       ├── webhooks.go         # Webhook endpoints
│   │       └── system.go           # System endpoints
│   │
│   └── parser/
│       └── mime.go                 # MIME message parser
│
├── web/                            # Web UI (Svelte/React)
│   ├── src/
│   │   ├── lib/
│   │   │   ├── api/                # API client
│   │   │   ├── components/         # UI components
│   │   │   └── stores/             # State management
│   │   ├── routes/
│   │   │   ├── +layout.svelte
│   │   │   ├── +page.svelte        # Dashboard
│   │   │   ├── login/
│   │   │   ├── inbox/
│   │   │   ├── message/
│   │   │   ├── users/
│   │   │   └── settings/
│   │   └── app.html
│   ├── static/
│   ├── package.json
│   ├── svelte.config.js
│   ├── tailwind.config.js
│   └── vite.config.js
│
├── webui/
│   └── embed.go                    # go:embed for Web UI
│
├── configs/
│   └── yunt.example.yaml           # Örnek konfigürasyon
│
├── scripts/
│   ├── build.sh                    # Build script
│   └── release.sh                  # Release script
│
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

---

## 4. Detaylı Özellikler

### 4.1 SMTP Server

#### 4.1.1 Özellik Listesi

| Özellik | Açıklama | Öncelik |
|---------|----------|---------|
| Mail Receiving | Gelen mailleri yakala ve kaydet | P0 |
| Multi-recipient | Birden fazla alıcıya mail | P0 |
| MIME Parsing | Attachment ve HTML body parsing | P0 |
| SMTP AUTH | PLAIN, LOGIN authentication | P1 |
| STARTTLS | TLS encryption desteği | P1 |
| Relay | Harici SMTP'ye iletme | P1 |
| Size Limit | Maksimum mesaj boyutu | P2 |
| Rate Limit | IP bazlı hız limiti | P2 |

#### 4.1.2 SMTP Akışı

```
┌──────────────┐                              ┌──────────────┐
│   Client     │                              │  Yunt SMTP   │
│ (App/Test)   │                              │   Server     │
└──────┬───────┘                              └──────┬───────┘
       │                                             │
       │  ──────── TCP Connect :1025 ──────────────▶ │
       │                                             │
       │  ◀─────── 220 yunt ESMTP ready ─────────── │
       │                                             │
       │  ──────── EHLO client.local ──────────────▶ │
       │                                             │
       │  ◀─────── 250-yunt Hello ─────────────────  │
       │           250-SIZE 26214400                 │
       │           250-AUTH PLAIN LOGIN              │
       │           250 STARTTLS                      │
       │                                             │
       │  ──────── AUTH PLAIN [credentials] ───────▶ │ (optional)
       │                                             │
       │  ◀─────── 235 Authentication successful ── │
       │                                             │
       │  ──────── MAIL FROM:<sender@test.com> ────▶ │
       │                                             │
       │  ◀─────── 250 OK ─────────────────────────  │
       │                                             │
       │  ──────── RCPT TO:<user@localhost> ───────▶ │
       │                                             │
       │  ◀─────── 250 OK ─────────────────────────  │
       │                                             │
       │  ──────── DATA ───────────────────────────▶ │
       │                                             │
       │  ◀─────── 354 Start mail input ─────────── │
       │                                             │
       │  ──────── [Email Content] ────────────────▶ │
       │           . (end)                           │
       │                                             │
       │  ◀─────── 250 OK: Message queued ─────────  │
       │                                             │
       │  ──────── QUIT ───────────────────────────▶ │
       │                                             │
       │  ◀─────── 221 Bye ─────────────────────────│
       │                                             │
       ▼                                             ▼
                        │
                        ▼
              ┌─────────────────┐
              │  Parse Message  │
              │  Store to DB    │
              │  Trigger Hooks  │
              │  (Relay if set) │
              └─────────────────┘
```

#### 4.1.3 Konfigürasyon

```go
type SMTPConfig struct {
    Host           string        `yaml:"host" env:"YUNT_SMTP_HOST" default:"0.0.0.0"`
    Port           int           `yaml:"port" env:"YUNT_SMTP_PORT" default:"1025"`
    Domain         string        `yaml:"domain" env:"YUNT_SMTP_DOMAIN" default:"localhost"`
    ReadTimeout    time.Duration `yaml:"read_timeout" default:"60s"`
    WriteTimeout   time.Duration `yaml:"write_timeout" default:"60s"`
    MaxMessageSize int64         `yaml:"max_message_size" default:"26214400"` // 25MB
    MaxRecipients  int           `yaml:"max_recipients" default:"100"`
    AuthRequired   bool          `yaml:"auth_required" default:"false"`
    TLS            TLSConfig     `yaml:"tls"`
    Relay          RelayConfig   `yaml:"relay"`
}

type RelayConfig struct {
    Enabled   bool     `yaml:"enabled" default:"false"`
    Host      string   `yaml:"host"`
    Port      int      `yaml:"port" default:"587"`
    Username  string   `yaml:"username"`
    Password  string   `yaml:"password"`
    UseTLS    bool     `yaml:"use_tls" default:"true"`
    AllowList []string `yaml:"allow_list"` // Sadece bu domainlere relay
}
```

### 4.2 IMAP Server

#### 4.2.1 Özellik Listesi

| Özellik | Açıklama | Öncelik |
|---------|----------|---------|
| IMAP4rev1 | RFC 3501 uyumlu | P0 |
| LOGIN | Authentication | P0 |
| LIST | Mailbox listesi | P0 |
| SELECT | Mailbox seçimi | P0 |
| FETCH | Mesaj getirme | P0 |
| STORE | Flag değiştirme | P0 |
| SEARCH | Mesaj arama | P1 |
| IDLE | Real-time push | P1 |
| STARTTLS | TLS desteği | P1 |
| COPY/MOVE | Mesaj taşıma | P2 |
| EXPUNGE | Mesaj silme | P0 |

#### 4.2.2 Desteklenen Flagler

```
\Seen      - Mesaj okundu
\Answered  - Mesaj yanıtlandı
\Flagged   - Yıldızlı/önemli
\Deleted   - Silinmek için işaretli
\Draft     - Taslak
```

#### 4.2.3 Varsayılan Mailboxlar

```
INBOX       - Gelen kutusu (otomatik oluşturulur)
Sent        - Gönderilenler
Drafts      - Taslaklar
Trash       - Çöp kutusu
Spam        - İstenmeyen
```

#### 4.2.4 Konfigürasyon

```go
type IMAPConfig struct {
    Host         string        `yaml:"host" env:"YUNT_IMAP_HOST" default:"0.0.0.0"`
    Port         int           `yaml:"port" env:"YUNT_IMAP_PORT" default:"1143"`
    ReadTimeout  time.Duration `yaml:"read_timeout" default:"60s"`
    WriteTimeout time.Duration `yaml:"write_timeout" default:"60s"`
    IdleTimeout  time.Duration `yaml:"idle_timeout" default:"30m"`
    TLS          TLSConfig     `yaml:"tls"`
}
```

### 4.3 REST API

#### 4.3.1 Endpoint Listesi

**Authentication**

| Method | Endpoint | Açıklama |
|--------|----------|----------|
| POST | `/api/v1/auth/login` | Kullanıcı girişi |
| POST | `/api/v1/auth/logout` | Çıkış |
| POST | `/api/v1/auth/refresh` | Token yenileme |
| GET | `/api/v1/auth/me` | Mevcut kullanıcı |

**Users (Admin)**

| Method | Endpoint | Açıklama |
|--------|----------|----------|
| GET | `/api/v1/users` | Kullanıcı listesi |
| POST | `/api/v1/users` | Yeni kullanıcı |
| GET | `/api/v1/users/:id` | Kullanıcı detayı |
| PUT | `/api/v1/users/:id` | Güncelleme |
| DELETE | `/api/v1/users/:id` | Silme |

**Mailboxes**

| Method | Endpoint | Açıklama |
|--------|----------|----------|
| GET | `/api/v1/mailboxes` | Mailbox listesi |
| POST | `/api/v1/mailboxes` | Yeni mailbox |
| GET | `/api/v1/mailboxes/:id` | Mailbox detayı |
| PUT | `/api/v1/mailboxes/:id` | Güncelleme |
| DELETE | `/api/v1/mailboxes/:id` | Silme |
| GET | `/api/v1/mailboxes/:id/stats` | İstatistikler |

**Messages**

| Method | Endpoint | Açıklama |
|--------|----------|----------|
| GET | `/api/v1/messages` | Mesaj listesi |
| GET | `/api/v1/messages/:id` | Mesaj detayı |
| DELETE | `/api/v1/messages/:id` | Mesaj sil |
| DELETE | `/api/v1/messages` | Toplu silme |
| GET | `/api/v1/messages/:id/raw` | Ham EML |
| GET | `/api/v1/messages/:id/html` | HTML body |
| GET | `/api/v1/messages/:id/text` | Plain text |
| PUT | `/api/v1/messages/:id/read` | Okundu işaretle |
| PUT | `/api/v1/messages/:id/unread` | Okunmadı |
| PUT | `/api/v1/messages/:id/star` | Yıldızla |
| PUT | `/api/v1/messages/:id/move` | Taşı |

**Attachments**

| Method | Endpoint | Açıklama |
|--------|----------|----------|
| GET | `/api/v1/messages/:id/attachments` | Ek listesi |
| GET | `/api/v1/messages/:id/attachments/:aid` | Ek indir |

**Search**

| Method | Endpoint | Açıklama |
|--------|----------|----------|
| GET | `/api/v1/search` | Basit arama |
| POST | `/api/v1/search/advanced` | Gelişmiş arama |

**Webhooks**

| Method | Endpoint | Açıklama |
|--------|----------|----------|
| GET | `/api/v1/webhooks` | Webhook listesi |
| POST | `/api/v1/webhooks` | Yeni webhook |
| PUT | `/api/v1/webhooks/:id` | Güncelleme |
| DELETE | `/api/v1/webhooks/:id` | Silme |
| POST | `/api/v1/webhooks/:id/test` | Test |

**System**

| Method | Endpoint | Açıklama |
|--------|----------|----------|
| GET | `/api/v1/health` | Health check |
| GET | `/api/v1/stats` | İstatistikler |
| DELETE | `/api/v1/system/messages` | Tüm mesajları sil |

#### 4.3.2 Response Format

**Başarılı Response**

```json
{
  "success": true,
  "data": {
    "id": "msg_123",
    "subject": "Test Email"
  },
  "meta": {
    "page": 1,
    "per_page": 50,
    "total": 1234,
    "total_pages": 25
  }
}
```

**Hata Response**

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid email format",
    "details": {
      "field": "email",
      "value": "invalid"
    }
  }
}
```

#### 4.3.3 Query Parameters

**Pagination**

```
GET /api/v1/messages?page=1&per_page=50
```

**Filtering**

```
GET /api/v1/messages?mailbox_id=xxx&unread=true&starred=false
GET /api/v1/messages?from=sender@test.com
GET /api/v1/messages?date_from=2024-01-01&date_to=2024-12-31
```

**Sorting**

```
GET /api/v1/messages?sort_by=received_at&order=desc
```

### 4.4 Web UI

#### 4.4.1 Sayfa Yapısı

```
┌─────────────────────────────────────────────────────────────────┐
│                         HEADER                                  │
│  ┌──────┐                              ┌─────┐ ┌─────────────┐  │
│  │ YUNT │                              │ 🔍  │ │ User ▾      │  │
│  └──────┘                              └─────┘ └─────────────┘  │
├─────────────────────────────────────────────────────────────────┤
│         │                                                       │
│ SIDEBAR │                    MAIN CONTENT                       │
│         │                                                       │
│ ┌─────┐ │  ┌─────────────────────────────────────────────────┐  │
│ │Inbox│ │  │                                                 │  │
│ │ (5) │ │  │              Message List /                     │  │
│ ├─────┤ │  │              Dashboard /                        │  │
│ │Sent │ │  │              Settings /                         │  │
│ │     │ │  │              etc.                               │  │
│ ├─────┤ │  │                                                 │  │
│ │Trash│ │  │                                                 │  │
│ │     │ │  │                                                 │  │
│ ├─────┤ │  │                                                 │  │
│ │Spam │ │  │                                                 │  │
│ │     │ │  │                                                 │  │
│ └─────┘ │  └─────────────────────────────────────────────────┘  │
│         │                                                       │
│ ─────── │                                                       │
│         │                                                       │
│ Users   │                                                       │
│ Settings│                                                       │
│         │                                                       │
└─────────────────────────────────────────────────────────────────┘
```

#### 4.4.2 Sayfalar

| Sayfa | URL | Açıklama |
|-------|-----|----------|
| Login | `/login` | Kullanıcı girişi |
| Dashboard | `/` | Genel bakış, istatistikler |
| Inbox | `/inbox` | Gelen kutusu |
| Message | `/message/:id` | Mesaj detayı |
| Users | `/users` | Kullanıcı yönetimi (admin) |
| Settings | `/settings` | Ayarlar |
| Webhooks | `/webhooks` | Webhook yönetimi |

#### 4.4.3 Dashboard Bileşenleri

```
┌─────────────────────────────────────────────────────────────────┐
│                         DASHBOARD                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌────────────┐   │
│  │   Total    │ │   Unread   │ │   Today    │ │ This Week  │   │
│  │   1,234    │ │     56     │ │     12     │ │    189     │   │
│  │  messages  │ │  messages  │ │  messages  │ │  messages  │   │
│  └────────────┘ └────────────┘ └────────────┘ └────────────┘   │
│                                                                 │
│  ┌─────────────────────────────┐ ┌─────────────────────────┐   │
│  │     Messages per Hour       │ │    Storage Usage        │   │
│  │     ▃▅▇█▆▄▂▃▅▇█▆▄          │ │    ████████░░ 78%       │   │
│  │                             │ │    156 MB / 200 MB      │   │
│  └─────────────────────────────┘ └─────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    Recent Messages                       │   │
│  ├─────────────────────────────────────────────────────────┤   │
│  │ ● sender@test.com - Welcome to our platform  - 2 min    │   │
│  │ ○ no-reply@app.co - Your order confirmed     - 15 min   │   │
│  │ ○ alerts@sys.io   - Server notification      - 1 hour   │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 4.4.4 Inbox View

```
┌─────────────────────────────────────────────────────────────────┐
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ 🔍 Search messages...                      [Filter ▾]   │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ☐ Select All    [Delete] [Mark Read] [Move ▾]    Showing 1-50 │
│  ───────────────────────────────────────────────────────────── │
│                                                                 │
│  ☐ ● ☆ │ sender@test.com      │ Welcome to Yunt      │ 2m     │
│  ☐ ○ ☆ │ no-reply@service.com │ Password reset req...│ 15m    │
│  ☐ ○ ★ │ alerts@monitor.io    │ CPU usage alert      │ 1h     │
│  ☐ ○ ☆ │ dev@company.com      │ New deployment sta...│ 2h     │
│  ☐ ○ ☆ │ test@localhost       │ Test email #42       │ 3h     │
│                                                                 │
│  ─────────────────────────────────────────────────────────────  │
│  │◀  1  2  3  4  5  ...  25  ▶│                                │
└─────────────────────────────────────────────────────────────────┘

Legend:
● = Unread
○ = Read
★ = Starred
☆ = Not starred
```

#### 4.4.5 Message Detail View

```
┌─────────────────────────────────────────────────────────────────┐
│  [← Back]                              [Delete] [Move ▾] [Raw]  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Subject: Welcome to Yunt - Your development mail server       │
│  ───────────────────────────────────────────────────────────── │
│                                                                 │
│  From:    sender@test.com                                       │
│  To:      developer@localhost                                   │
│  CC:      team@localhost                                        │
│  Date:    December 24, 2024, 10:30 AM                          │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │  [HTML]  [Text]  [Headers]  [Raw]                         │ │
│  ├───────────────────────────────────────────────────────────┤ │
│  │                                                           │ │
│  │  Hello Developer,                                         │ │
│  │                                                           │ │
│  │  Welcome to Yunt! Your mail server is now ready.          │ │
│  │                                                           │ │
│  │  SMTP: localhost:1025                                     │ │
│  │  IMAP: localhost:1143                                     │ │
│  │  Web:  localhost:8025                                     │ │
│  │                                                           │ │
│  │  Happy testing!                                           │ │
│  │                                                           │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
│  Attachments (2):                                               │
│  ┌──────────────────┐ ┌──────────────────┐                     │
│  │ 📎 config.yaml   │ │ 📎 readme.pdf    │                     │
│  │    2.3 KB        │ │    156 KB        │                     │
│  │   [Download]     │ │   [Download]     │                     │
│  └──────────────────┘ └──────────────────┘                     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 4.4.6 Teknoloji Stack

| Katman | Teknoloji |
|--------|-----------|
| Framework | Svelte 5 (SvelteKit) |
| Styling | Tailwind CSS |
| Icons | Lucide Icons |
| State | Svelte Stores |
| HTTP | Fetch API |
| Build | Vite |
| Embed | go:embed |

---

## 5. Domain Models

### 5.1 User

```go
type User struct {
    ID           string    `json:"id"`
    Username     string    `json:"username"`
    Email        string    `json:"email"`
    PasswordHash string    `json:"-"`
    DisplayName  string    `json:"display_name"`
    Role         Role      `json:"role"`
    IsActive     bool      `json:"is_active"`
    LastLoginAt  *time.Time `json:"last_login_at"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}

type Role string

const (
    RoleAdmin Role = "admin"
    RoleUser  Role = "user"
)
```

### 5.2 Mailbox

```go
type Mailbox struct {
    ID           string      `json:"id"`
    UserID       string      `json:"user_id"`
    Name         string      `json:"name"`
    Slug         string      `json:"slug"`
    Type         MailboxType `json:"type"`
    SortOrder    int         `json:"sort_order"`
    MessageCount int64       `json:"message_count"`
    UnreadCount  int64       `json:"unread_count"`
    CreatedAt    time.Time   `json:"created_at"`
    UpdatedAt    time.Time   `json:"updated_at"`
}

type MailboxType string

const (
    MailboxTypeSystem MailboxType = "system" // INBOX, Sent, etc.
    MailboxTypeCustom MailboxType = "custom" // User-created
)

type MailboxStats struct {
    TotalCount  int64 `json:"total_count"`
    UnreadCount int64 `json:"unread_count"`
    StarredCount int64 `json:"starred_count"`
    TotalSize   int64 `json:"total_size"`
}
```

### 5.3 Message

```go
type Message struct {
    ID          string    `json:"id"`
    MailboxID   string    `json:"mailbox_id"`
    UserID      string    `json:"user_id"`
    
    // RFC 5322 Headers
    MessageID   string    `json:"message_id"`
    InReplyTo   string    `json:"in_reply_to,omitempty"`
    References  []string  `json:"references,omitempty"`
    
    // Addresses
    From        Address   `json:"from"`
    To          []Address `json:"to"`
    CC          []Address `json:"cc,omitempty"`
    BCC         []Address `json:"bcc,omitempty"`
    ReplyTo     []Address `json:"reply_to,omitempty"`
    
    // Content
    Subject     string    `json:"subject"`
    TextBody    string    `json:"text_body,omitempty"`
    HTMLBody    string    `json:"html_body,omitempty"`
    
    // Raw storage
    RawMessage  []byte    `json:"-"`
    Size        int64     `json:"size"`
    
    // Flags
    IsRead      bool      `json:"is_read"`
    IsStarred   bool      `json:"is_starred"`
    IsDeleted   bool      `json:"is_deleted"`
    
    // Metadata
    HasAttachments bool      `json:"has_attachments"`
    AttachmentCount int      `json:"attachment_count"`
    ReceivedAt  time.Time `json:"received_at"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type Address struct {
    Name    string `json:"name,omitempty"`
    Address string `json:"address"`
}
```

### 5.4 Attachment

```go
type Attachment struct {
    ID          string    `json:"id"`
    MessageID   string    `json:"message_id"`
    Filename    string    `json:"filename"`
    ContentType string    `json:"content_type"`
    Size        int64     `json:"size"`
    ContentID   string    `json:"content_id,omitempty"` // For inline
    IsInline    bool      `json:"is_inline"`
    Content     []byte    `json:"-"`
    CreatedAt   time.Time `json:"created_at"`
}
```

### 5.5 Webhook

```go
type Webhook struct {
    ID        string            `json:"id"`
    Name      string            `json:"name"`
    URL       string            `json:"url"`
    Secret    string            `json:"secret,omitempty"`
    Events    []WebhookEvent    `json:"events"`
    Headers   map[string]string `json:"headers,omitempty"`
    IsActive  bool              `json:"is_active"`
    CreatedAt time.Time         `json:"created_at"`
    UpdatedAt time.Time         `json:"updated_at"`
}

type WebhookEvent string

const (
    WebhookEventMessageReceived WebhookEvent = "message.received"
    WebhookEventMessageDeleted  WebhookEvent = "message.deleted"
    WebhookEventUserCreated     WebhookEvent = "user.created"
)
```

---

## 6. Repository Layer

### 6.1 Interface Definition

```go
package repository

import (
    "context"
    "yunt/internal/domain"
)

// Repository - Ana repository interface
type Repository interface {
    // Lifecycle
    Connect(ctx context.Context) error
    Close() error
    Ping(ctx context.Context) error
    Migrate(ctx context.Context) error
    
    // Sub-repositories
    Users() UserRepository
    Mailboxes() MailboxRepository
    Messages() MessageRepository
    Attachments() AttachmentRepository
    Webhooks() WebhookRepository
    Settings() SettingsRepository
}

// UserRepository - Kullanıcı işlemleri
type UserRepository interface {
    Create(ctx context.Context, user *domain.User) error
    GetByID(ctx context.Context, id string) (*domain.User, error)
    GetByEmail(ctx context.Context, email string) (*domain.User, error)
    GetByUsername(ctx context.Context, username string) (*domain.User, error)
    List(ctx context.Context, opts ListOptions) ([]*domain.User, int64, error)
    Update(ctx context.Context, user *domain.User) error
    Delete(ctx context.Context, id string) error
    UpdateLastLogin(ctx context.Context, id string) error
}

// MailboxRepository - Mailbox işlemleri
type MailboxRepository interface {
    Create(ctx context.Context, mailbox *domain.Mailbox) error
    GetByID(ctx context.Context, id string) (*domain.Mailbox, error)
    GetBySlug(ctx context.Context, userID, slug string) (*domain.Mailbox, error)
    ListByUser(ctx context.Context, userID string) ([]*domain.Mailbox, error)
    Update(ctx context.Context, mailbox *domain.Mailbox) error
    Delete(ctx context.Context, id string) error
    GetStats(ctx context.Context, id string) (*domain.MailboxStats, error)
    CreateDefaultMailboxes(ctx context.Context, userID string) error
}

// MessageRepository - Mesaj işlemleri
type MessageRepository interface {
    Create(ctx context.Context, msg *domain.Message) error
    GetByID(ctx context.Context, id string) (*domain.Message, error)
    List(ctx context.Context, opts MessageListOptions) ([]*domain.Message, int64, error)
    Update(ctx context.Context, msg *domain.Message) error
    Delete(ctx context.Context, id string) error
    DeleteByMailbox(ctx context.Context, mailboxID string) error
    DeleteAll(ctx context.Context, userID string) error
    
    // Raw message
    GetRaw(ctx context.Context, id string) ([]byte, error)
    
    // Flags
    MarkAsRead(ctx context.Context, id string) error
    MarkAsUnread(ctx context.Context, id string) error
    ToggleStar(ctx context.Context, id string) error
    
    // Move
    MoveToMailbox(ctx context.Context, msgID, mailboxID string) error
    
    // Search
    Search(ctx context.Context, query SearchQuery) ([]*domain.Message, int64, error)
    
    // Stats
    GetStats(ctx context.Context, userID string) (*domain.MessageStats, error)
}

// AttachmentRepository - Attachment işlemleri
type AttachmentRepository interface {
    Create(ctx context.Context, att *domain.Attachment) error
    GetByID(ctx context.Context, id string) (*domain.Attachment, error)
    ListByMessage(ctx context.Context, messageID string) ([]*domain.Attachment, error)
    GetContent(ctx context.Context, id string) ([]byte, error)
    Delete(ctx context.Context, id string) error
    DeleteByMessage(ctx context.Context, messageID string) error
}

// WebhookRepository - Webhook işlemleri
type WebhookRepository interface {
    Create(ctx context.Context, webhook *domain.Webhook) error
    GetByID(ctx context.Context, id string) (*domain.Webhook, error)
    List(ctx context.Context) ([]*domain.Webhook, error)
    ListActive(ctx context.Context) ([]*domain.Webhook, error)
    Update(ctx context.Context, webhook *domain.Webhook) error
    Delete(ctx context.Context, id string) error
}

// SettingsRepository - Ayar işlemleri
type SettingsRepository interface {
    Get(ctx context.Context, key string) (string, error)
    Set(ctx context.Context, key, value string) error
    GetAll(ctx context.Context) (map[string]string, error)
    Delete(ctx context.Context, key string) error
}

// ListOptions - Pagination ve sıralama
type ListOptions struct {
    Page    int    `query:"page"`
    PerPage int    `query:"per_page"`
    SortBy  string `query:"sort_by"`
    Order   string `query:"order"` // asc, desc
}

// MessageListOptions - Mesaj listesi filtreleri
type MessageListOptions struct {
    ListOptions
    MailboxID string     `query:"mailbox_id"`
    UserID    string     `query:"-"`
    Unread    *bool      `query:"unread"`
    Starred   *bool      `query:"starred"`
    From      string     `query:"from"`
    To        string     `query:"to"`
    Subject   string     `query:"subject"`
    DateFrom  *time.Time `query:"date_from"`
    DateTo    *time.Time `query:"date_to"`
}

// SearchQuery - Arama sorgusu
type SearchQuery struct {
    Term           string     `json:"term"`
    From           string     `json:"from"`
    To             string     `json:"to"`
    Subject        string     `json:"subject"`
    Body           string     `json:"body"`
    HasAttachments *bool      `json:"has_attachments"`
    DateFrom       *time.Time `json:"date_from"`
    DateTo         *time.Time `json:"date_to"`
    MailboxID      string     `json:"mailbox_id"`
    UserID         string     `json:"-"`
    Page           int        `json:"page"`
    PerPage        int        `json:"per_page"`
}
```

### 6.2 Repository Factory

```go
package repository

import (
    "fmt"
    "yunt/internal/config"
    "yunt/internal/repository/mongodb"
    "yunt/internal/repository/mysql"
    "yunt/internal/repository/postgres"
    "yunt/internal/repository/sqlite"
)

func New(cfg *config.DatabaseConfig) (Repository, error) {
    switch cfg.Driver {
    case "sqlite":
        return sqlite.New(cfg.SQLite)
    case "postgres":
        return postgres.New(cfg.Postgres)
    case "mysql":
        return mysql.New(cfg.MySQL)
    case "mongodb":
        return mongodb.New(cfg.MongoDB)
    default:
        return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
    }
}
```

---

## 7. Database Schemas

### 7.1 SQL Schema (PostgreSQL/MySQL/SQLite)

```sql
-- Users table
CREATE TABLE users (
    id            VARCHAR(36) PRIMARY KEY,
    username      VARCHAR(100) NOT NULL UNIQUE,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    display_name  VARCHAR(255),
    role          VARCHAR(20) NOT NULL DEFAULT 'user',
    is_active     BOOLEAN NOT NULL DEFAULT true,
    last_login_at TIMESTAMP NULL,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username);

-- Mailboxes table
CREATE TABLE mailboxes (
    id         VARCHAR(36) PRIMARY KEY,
    user_id    VARCHAR(36) NOT NULL,
    name       VARCHAR(100) NOT NULL,
    slug       VARCHAR(100) NOT NULL,
    type       VARCHAR(20) NOT NULL DEFAULT 'custom',
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(user_id, slug)
);

CREATE INDEX idx_mailboxes_user ON mailboxes(user_id);

-- Messages table
CREATE TABLE messages (
    id              VARCHAR(36) PRIMARY KEY,
    mailbox_id      VARCHAR(36) NOT NULL,
    user_id         VARCHAR(36) NOT NULL,
    message_id      VARCHAR(255),
    in_reply_to     VARCHAR(255),
    references_list TEXT,
    from_name       VARCHAR(255),
    from_address    VARCHAR(255) NOT NULL,
    to_list         TEXT NOT NULL,
    cc_list         TEXT,
    bcc_list        TEXT,
    reply_to_list   TEXT,
    subject         VARCHAR(1000),
    text_body       TEXT,
    html_body       TEXT,
    raw_message     LONGBLOB,
    size            BIGINT NOT NULL DEFAULT 0,
    is_read         BOOLEAN NOT NULL DEFAULT false,
    is_starred      BOOLEAN NOT NULL DEFAULT false,
    is_deleted      BOOLEAN NOT NULL DEFAULT false,
    has_attachments BOOLEAN NOT NULL DEFAULT false,
    attachment_count INT NOT NULL DEFAULT 0,
    received_at     TIMESTAMP NOT NULL,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (mailbox_id) REFERENCES mailboxes(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_messages_mailbox ON messages(mailbox_id);
CREATE INDEX idx_messages_user ON messages(user_id);
CREATE INDEX idx_messages_received ON messages(received_at DESC);
CREATE INDEX idx_messages_from ON messages(from_address);
CREATE INDEX idx_messages_read ON messages(is_read);
CREATE INDEX idx_messages_starred ON messages(is_starred);

-- Full-text search (PostgreSQL)
-- CREATE INDEX idx_messages_search ON messages 
--     USING gin(to_tsvector('english', subject || ' ' || COALESCE(text_body, '')));

-- Attachments table
CREATE TABLE attachments (
    id           VARCHAR(36) PRIMARY KEY,
    message_id   VARCHAR(36) NOT NULL,
    filename     VARCHAR(255) NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    size         BIGINT NOT NULL,
    content_id   VARCHAR(255),
    is_inline    BOOLEAN NOT NULL DEFAULT false,
    content      LONGBLOB NOT NULL,
    created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE
);

CREATE INDEX idx_attachments_message ON attachments(message_id);

-- Webhooks table
CREATE TABLE webhooks (
    id         VARCHAR(36) PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,
    url        VARCHAR(500) NOT NULL,
    secret     VARCHAR(255),
    events     TEXT NOT NULL,
    headers    TEXT,
    is_active  BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Settings table
CREATE TABLE settings (
    key        VARCHAR(100) PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### 7.2 MongoDB Schema

```javascript
// users collection
{
  _id: ObjectId,
  username: { type: String, unique: true, required: true },
  email: { type: String, unique: true, required: true },
  password_hash: { type: String, required: true },
  display_name: String,
  role: { type: String, enum: ['admin', 'user'], default: 'user' },
  is_active: { type: Boolean, default: true },
  last_login_at: Date,
  created_at: { type: Date, default: Date.now },
  updated_at: { type: Date, default: Date.now }
}

// Indexes
db.users.createIndex({ email: 1 }, { unique: true })
db.users.createIndex({ username: 1 }, { unique: true })

// mailboxes collection
{
  _id: ObjectId,
  user_id: { type: ObjectId, ref: 'users', required: true },
  name: { type: String, required: true },
  slug: { type: String, required: true },
  type: { type: String, enum: ['system', 'custom'], default: 'custom' },
  sort_order: { type: Number, default: 0 },
  created_at: { type: Date, default: Date.now },
  updated_at: { type: Date, default: Date.now }
}

// Indexes
db.mailboxes.createIndex({ user_id: 1 })
db.mailboxes.createIndex({ user_id: 1, slug: 1 }, { unique: true })

// messages collection
{
  _id: ObjectId,
  mailbox_id: { type: ObjectId, ref: 'mailboxes', required: true },
  user_id: { type: ObjectId, ref: 'users', required: true },
  message_id: String,
  in_reply_to: String,
  references: [String],
  from: {
    name: String,
    address: { type: String, required: true }
  },
  to: [{
    name: String,
    address: { type: String, required: true }
  }],
  cc: [{
    name: String,
    address: String
  }],
  bcc: [{
    name: String,
    address: String
  }],
  reply_to: [{
    name: String,
    address: String
  }],
  subject: String,
  text_body: String,
  html_body: String,
  raw_message: Buffer,
  size: { type: Number, default: 0 },
  is_read: { type: Boolean, default: false },
  is_starred: { type: Boolean, default: false },
  is_deleted: { type: Boolean, default: false },
  has_attachments: { type: Boolean, default: false },
  attachment_count: { type: Number, default: 0 },
  received_at: { type: Date, required: true },
  created_at: { type: Date, default: Date.now },
  updated_at: { type: Date, default: Date.now }
}

// Indexes
db.messages.createIndex({ mailbox_id: 1 })
db.messages.createIndex({ user_id: 1 })
db.messages.createIndex({ received_at: -1 })
db.messages.createIndex({ "from.address": 1 })
db.messages.createIndex({ is_read: 1 })
db.messages.createIndex({ is_starred: 1 })
db.messages.createIndex({ 
  subject: "text", 
  text_body: "text" 
}, { name: "messages_text_search" })

// attachments collection
{
  _id: ObjectId,
  message_id: { type: ObjectId, ref: 'messages', required: true },
  filename: { type: String, required: true },
  content_type: { type: String, required: true },
  size: { type: Number, required: true },
  content_id: String,
  is_inline: { type: Boolean, default: false },
  content: { type: Buffer, required: true },
  created_at: { type: Date, default: Date.now }
}

// Indexes
db.attachments.createIndex({ message_id: 1 })

// webhooks collection
{
  _id: ObjectId,
  name: { type: String, required: true },
  url: { type: String, required: true },
  secret: String,
  events: [{ type: String }],
  headers: Object,
  is_active: { type: Boolean, default: true },
  created_at: { type: Date, default: Date.now },
  updated_at: { type: Date, default: Date.now }
}

// settings collection
{
  _id: String, // key
  value: Mixed,
  updated_at: { type: Date, default: Date.now }
}
```

---

## 8. Configuration

### 8.1 Config File (yunt.yaml)

```yaml
# =============================================================================
# YUNT CONFIGURATION
# =============================================================================

# Server Settings
server:
  name: "Yunt Mail Server"
  env: "development"  # development, production

# -----------------------------------------------------------------------------
# SMTP Server Configuration
# -----------------------------------------------------------------------------
smtp:
  host: "0.0.0.0"
  port: 1025
  domain: "localhost"
  read_timeout: "60s"
  write_timeout: "60s"
  max_message_size: 26214400  # 25MB
  max_recipients: 100
  auth_required: false
  
  tls:
    enabled: false
    cert_file: ""
    key_file: ""
  
  relay:
    enabled: false
    host: "smtp.gmail.com"
    port: 587
    username: ""
    password: ""
    use_tls: true
    allow_list: []  # Empty = allow all, otherwise only these domains

# -----------------------------------------------------------------------------
# IMAP Server Configuration
# -----------------------------------------------------------------------------
imap:
  host: "0.0.0.0"
  port: 1143
  read_timeout: "60s"
  write_timeout: "60s"
  idle_timeout: "30m"
  
  tls:
    enabled: false
    cert_file: ""
    key_file: ""

# -----------------------------------------------------------------------------
# API Server Configuration
# -----------------------------------------------------------------------------
api:
  host: "0.0.0.0"
  port: 8025
  read_timeout: "30s"
  write_timeout: "30s"
  
  cors:
    enabled: true
    origins:
      - "*"
    methods:
      - "GET"
      - "POST"
      - "PUT"
      - "DELETE"
      - "OPTIONS"
    headers:
      - "Authorization"
      - "Content-Type"

# -----------------------------------------------------------------------------
# Authentication Configuration
# -----------------------------------------------------------------------------
auth:
  jwt_secret: "change-me-in-production-use-strong-random-string"
  jwt_expiry: "24h"
  refresh_expiry: "168h"  # 7 days
  bcrypt_cost: 10

# -----------------------------------------------------------------------------
# Database Configuration
# -----------------------------------------------------------------------------
database:
  driver: "sqlite"  # sqlite, postgres, mysql, mongodb
  
  sqlite:
    path: "./data/yunt.db"
    journal_mode: "WAL"
  
  postgres:
    host: "localhost"
    port: 5432
    username: "yunt"
    password: "yunt"
    database: "yunt"
    sslmode: "disable"
    max_open_conns: 25
    max_idle_conns: 5
    conn_max_lifetime: "5m"
  
  mysql:
    host: "localhost"
    port: 3306
    username: "yunt"
    password: "yunt"
    database: "yunt"
    charset: "utf8mb4"
    parse_time: true
    max_open_conns: 25
    max_idle_conns: 5
    conn_max_lifetime: "5m"
  
  mongodb:
    uri: "mongodb://localhost:27017"
    database: "yunt"
    timeout: "10s"

# -----------------------------------------------------------------------------
# Storage Configuration
# -----------------------------------------------------------------------------
storage:
  # Where to store attachments (if not in DB)
  attachments_path: "./data/attachments"
  max_attachment_size: 26214400  # 25MB

# -----------------------------------------------------------------------------
# Logging Configuration
# -----------------------------------------------------------------------------
logging:
  level: "info"      # debug, info, warn, error
  format: "text"     # text, json
  output: "stdout"   # stdout, file
  file_path: "./logs/yunt.log"
  
# -----------------------------------------------------------------------------
# Default Admin User (created on first run)
# -----------------------------------------------------------------------------
admin:
  username: "admin"
  email: "admin@localhost"
  password: "admin123"  # CHANGE THIS IN PRODUCTION!
```

### 8.2 Environment Variables

Tüm konfigürasyon değerleri environment variable ile override edilebilir:

```bash
# Server
YUNT_SERVER_ENV=production

# SMTP
YUNT_SMTP_HOST=0.0.0.0
YUNT_SMTP_PORT=1025
YUNT_SMTP_AUTH_REQUIRED=true

# IMAP
YUNT_IMAP_HOST=0.0.0.0
YUNT_IMAP_PORT=1143

# API
YUNT_API_HOST=0.0.0.0
YUNT_API_PORT=8025

# Database
YUNT_DATABASE_DRIVER=postgres
YUNT_DATABASE_POSTGRES_HOST=db.example.com
YUNT_DATABASE_POSTGRES_PORT=5432
YUNT_DATABASE_POSTGRES_USERNAME=yunt
YUNT_DATABASE_POSTGRES_PASSWORD=secret
YUNT_DATABASE_POSTGRES_DATABASE=yunt

# Auth
YUNT_AUTH_JWT_SECRET=super-secret-key-change-me

# Logging
YUNT_LOGGING_LEVEL=info
YUNT_LOGGING_FORMAT=json
```

---

## 9. CLI Commands

```bash
# Sunucuyu başlat
yunt serve

# Belirli config dosyası ile başlat
yunt serve --config /path/to/yunt.yaml

# Sadece belirli servisleri başlat
yunt serve --smtp --imap          # Web UI olmadan
yunt serve --api                  # Sadece Web UI/API

# Veritabanı migration
yunt migrate

# Admin kullanıcı oluştur
yunt user create --username admin --email admin@localhost --password secret --role admin

# Kullanıcı listele
yunt user list

# Kullanıcı sil
yunt user delete --username testuser

# Tüm mesajları sil
yunt messages delete-all --confirm

# Versiyon bilgisi
yunt version

# Health check
yunt health
```

---

## 10. Docker Support

### 10.1 Dockerfile

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache gcc musl-dev nodejs npm git

WORKDIR /app

# Go dependencies
COPY go.mod go.sum ./
RUN go mod download

# Frontend build
COPY web/ ./web/
WORKDIR /app/web
RUN npm ci && npm run build

# Go build
WORKDIR /app
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-s -w -X main.Version=$(git describe --tags --always)" \
    -o yunt ./cmd/yunt

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/yunt .
COPY --from=builder /app/configs/yunt.example.yaml ./yunt.yaml

RUN mkdir -p /app/data /app/logs && \
    adduser -D -H yunt && \
    chown -R yunt:yunt /app

USER yunt

EXPOSE 1025 1143 8025

VOLUME ["/app/data", "/app/logs"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:8025/api/v1/health || exit 1

ENTRYPOINT ["./yunt"]
CMD ["serve"]
```

### 10.2 Docker Compose

```yaml
version: '3.8'

services:
  yunt:
    build: .
    container_name: yunt
    ports:
      - "1025:1025"   # SMTP
      - "1143:1143"   # IMAP
      - "8025:8025"   # Web UI & API
    volumes:
      - yunt-data:/app/data
      - ./yunt.yaml:/app/yunt.yaml:ro
    environment:
      - YUNT_SERVER_ENV=production
      - YUNT_AUTH_JWT_SECRET=${JWT_SECRET:-change-me}
    restart: unless-stopped
    networks:
      - yunt-network

  # PostgreSQL (optional)
  postgres:
    image: postgres:16-alpine
    container_name: yunt-postgres
    environment:
      POSTGRES_USER: yunt
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-yunt}
      POSTGRES_DB: yunt
    volumes:
      - postgres-data:/var/lib/postgresql/data
    restart: unless-stopped
    networks:
      - yunt-network
    profiles:
      - postgres

  # MySQL (optional)
  mysql:
    image: mysql:8.0
    container_name: yunt-mysql
    environment:
      MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD:-root}
      MYSQL_USER: yunt
      MYSQL_PASSWORD: ${MYSQL_PASSWORD:-yunt}
      MYSQL_DATABASE: yunt
    volumes:
      - mysql-data:/var/lib/mysql
    restart: unless-stopped
    networks:
      - yunt-network
    profiles:
      - mysql

  # MongoDB (optional)
  mongodb:
    image: mongo:7
    container_name: yunt-mongodb
    environment:
      MONGO_INITDB_DATABASE: yunt
    volumes:
      - mongodb-data:/data/db
    restart: unless-stopped
    networks:
      - yunt-network
    profiles:
      - mongodb

volumes:
  yunt-data:
  postgres-data:
  mysql-data:
  mongodb-data:

networks:
  yunt-network:
    driver: bridge
```

### 10.3 Docker Commands

```bash
# SQLite ile başlat (varsayılan)
docker-compose up -d yunt

# PostgreSQL ile başlat
docker-compose --profile postgres up -d

# MySQL ile başlat
docker-compose --profile mysql up -d

# MongoDB ile başlat
docker-compose --profile mongodb up -d

# Logları izle
docker-compose logs -f yunt

# Durdur
docker-compose down
```

---

## 11. Development Roadmap

### Phase 1: Foundation (2-3 hafta)

- [ ] Proje yapısı ve Go modules
- [ ] Config loader (YAML + ENV)
- [ ] Domain models
- [ ] Repository interface
- [ ] SQLite repository implementasyonu
- [ ] Temel SMTP server (mail yakalama)
- [ ] Temel API server (health, basit endpoints)
- [ ] CLI framework

### Phase 2: Core Protocols (2-3 hafta)

- [ ] SMTP authentication
- [ ] SMTP relay support
- [ ] SMTP TLS support
- [ ] IMAP server implementasyonu
- [ ] IMAP IDLE support
- [ ] IMAP TLS support
- [ ] MIME parser (attachments, HTML)

### Phase 3: Multi-Database (2 hafta)

- [ ] PostgreSQL repository
- [ ] MySQL repository
- [ ] MongoDB repository
- [ ] Repository factory
- [ ] Migration system

### Phase 4: API & Auth (1-2 hafta)

- [ ] JWT authentication
- [ ] User management endpoints
- [ ] Mailbox endpoints
- [ ] Message endpoints
- [ ] Search endpoints
- [ ] Webhook system
- [ ] Rate limiting

### Phase 5: Web UI (2-3 hafta)

- [ ] Svelte project setup
- [ ] Login page
- [ ] Dashboard
- [ ] Inbox view
- [ ] Message detail
- [ ] User management
- [ ] Settings page
- [ ] Real-time updates (WebSocket)

### Phase 6: Polish & Release (1 hafta)

- [ ] Web UI embed (go:embed)
- [ ] Docker support
- [ ] Documentation
- [ ] Unit tests
- [ ] Integration tests
- [ ] Performance optimization
- [ ] Release builds

---

## 12. Go Dependencies

```go
// go.mod
module yunt

go 1.22

require (
    // SMTP
    github.com/emersion/go-smtp v0.21.0
    github.com/emersion/go-sasl v0.0.0-20231106173351-e73c9f7bad43
    
    // IMAP
    github.com/emersion/go-imap/v2 v2.0.0-beta.1
    
    // Mail parsing
    github.com/emersion/go-message v0.18.0
    github.com/jhillyerd/enmime v1.2.0
    
    // HTTP Router
    github.com/labstack/echo/v4 v4.11.4
    
    // Database - SQL
    github.com/jmoiron/sqlx v1.3.5
    github.com/mattn/go-sqlite3 v1.14.22
    github.com/go-sql-driver/mysql v1.7.1
    github.com/lib/pq v1.10.9
    
    // Database - MongoDB
    go.mongodb.org/mongo-driver v1.13.1
    
    // Auth
    github.com/golang-jwt/jwt/v5 v5.2.0
    golang.org/x/crypto v0.18.0
    
    // Config
    github.com/spf13/viper v1.18.2
    github.com/spf13/cobra v1.8.0
    
    // Logging
    github.com/rs/zerolog v1.31.0
    
    // Utils
    github.com/google/uuid v1.6.0
    github.com/gorilla/websocket v1.5.1
)
```

---

## 13. Security Considerations

| Alan | Önlem |
|------|-------|
| Authentication | JWT tokens, bcrypt password hashing |
| Authorization | Role-based access (admin/user) |
| Transport | TLS support for SMTP, IMAP, HTTP |
| Input Validation | Tüm API endpoints için validation |
| XSS | HTML content sanitization |
| CSRF | SameSite cookies, CORS |
| Rate Limiting | IP ve user bazlı |
| Secrets | Environment variables, no hardcoding |

---

## 14. Performance Targets

| Metrik | Hedef |
|--------|-------|
| SMTP throughput | 100+ msg/sec |
| API response (p95) | < 50ms |
| Memory (idle) | < 50MB |
| Memory (10k msgs) | < 200MB |
| Startup time | < 2s |
| DB query time | < 10ms |

---

## 15. Success Metrics

| Metrik | Hedef |
|--------|-------|
| Test coverage | > 70% |
| Documentation | Complete |
| Docker image size | < 50MB |
| Zero critical bugs | ✓ |
| Multi-DB parity | 100% |

---

## Changelog

| Versiyon | Tarih | Değişiklikler |
|----------|-------|---------------|
| 1.0 | December 2024 | Initial PRD |
