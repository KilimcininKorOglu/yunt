# Yunt Eksiklik Raporu

Task dosyaları (F001-F008 + PRD) ile mevcut kod tabanının karşılaştırmalı analizi.

---

## F001: Project Foundation & Configuration

| Task   | Açıklama                                | Durum         | Detay                                                                         |
|--------|----------------------------------------|---------------|-------------------------------------------------------------------------------|
| T001   | Go module ve proje yapısı              | Tamamlandı    | `go.mod`, `cmd/yunt/`, `internal/` yapısı mevcut.                             |
| T002   | Configuration yapısı ve loader         | Tamamlandı    | `internal/config/` — Config struct, Viper loader, validation, defaults.       |
| T003   | Zerolog logger                         | Tamamlandı    | `internal/config/logger.go` mevcut.                                           |
| T004   | CLI framework (Cobra)                  | Tamamlandı    | `cmd/yunt/cmd_root.go` + tüm subcommand'lar mevcut.                          |
| T005   | Makefile                               | Tamamlandı    | Build, test, lint, fmt, web komutları mevcut.                                 |
| T006   | Core dependencies                      | Tamamlandı    | go.mod'da tüm bağımlılıklar var.                                             |

**F001 Durumu: TAMAMLANDI**

---

## F002: Domain Models & Repository Layer

| Task   | Açıklama                                | Durum         | Detay                                                                         |
|--------|----------------------------------------|---------------|-------------------------------------------------------------------------------|
| T007   | User domain modeli                     | Tamamlandı    | `internal/domain/user.go` + test mevcut.                                      |
| T008   | Mailbox domain modeli                  | Tamamlandı    | `internal/domain/mailbox.go` + test mevcut.                                   |
| T009   | Message domain modeli                  | Tamamlandı    | `internal/domain/message.go` + test mevcut.                                   |
| T010   | Repository interface'leri              | Tamamlandı    | `internal/repository/repository.go` aggregate interface mevcut.               |
| T011   | SQLite implementasyonu                 | Tamamlandı    | `internal/repository/sqlite/` — tüm repository'ler mevcut.                   |
| T012   | Migration sistemi                      | Tamamlandı    | `internal/repository/sqlite/migrations.go` + SQL dosyaları.                   |
| T013   | Repository testleri                    | Kısmen        | `sqlite_test.go`, `integration_test.go` mevcut. Pagination ve stats testlerinde önceden var olan hatalar. |

**F002 Durumu: %95 TAMAMLANDI** — SQLite pagination off-by-one ve stats scan hataları.

---

## F003: SMTP Server

| Task   | Açıklama                                | Durum         | Detay                                                                         |
|--------|----------------------------------------|---------------|-------------------------------------------------------------------------------|
| T014   | SMTP session handler                   | Tamamlandı    | `internal/smtp/session.go` mevcut.                                            |
| T015   | SMTP backend                           | Tamamlandı    | `internal/smtp/backend.go` mevcut.                                            |
| T016   | MIME parser                            | Tamamlandı    | `internal/parser/mime.go`, `address.go`, `attachment.go` mevcut.              |
| T017   | SMTP server lifecycle                  | Tamamlandı    | `internal/smtp/server.go` — Start/Stop mevcut.                               |
| T018   | STARTTLS desteği                       | Tamamlandı    | `internal/smtp/tls.go`, `config.go` — TLS config ve STARTTLS mevcut.         |
| T019   | SMTP relay                             | Tamamlandı    | `internal/service/relay.go` + `smtp/session.go:relayMessage()`.              |
| T020   | Rate limiting                          | Tamamlandı    | `internal/smtp/ratelimit.go` + test mevcut.                                  |
| T021   | SMTP testleri                          | Kısmen        | 7 test dosyası mevcut. `ratelimit_test.go`'da flaky test var.                |

**F003 Durumu: %95 TAMAMLANDI** — Flaky rate limit testi.

---

## F004: IMAP Server

| Task   | Açıklama                                | Durum         | Detay                                                                         |
|--------|----------------------------------------|---------------|-------------------------------------------------------------------------------|
| T022   | IMAP backend                           | Tamamlandı    | `internal/imap/backend.go` mevcut.                                            |
| T023   | Mailbox operations                     | Tamamlandı    | `mailbox.go`, `mailbox_ops.go`, `mailbox_list.go` mevcut.                    |
| T024   | FETCH komutu                           | Tamamlandı    | `internal/imap/fetch.go` + test.                                              |
| T025   | STORE komutu                           | Tamamlandı    | `internal/imap/store.go` + test.                                              |
| T026   | SEARCH komutu                          | Tamamlandı    | `internal/imap/search.go` + `search_parser.go` + test.                       |
| T027   | COPY/EXPUNGE                           | Tamamlandı    | `copy.go`, `expunge.go` + testler.                                            |
| T028   | IDLE real-time notifications           | Tamamlandı    | `internal/imap/idle.go`, `notify.go` mevcut.                                 |
| T029   | IMAP TLS                               | Tamamlandı    | `internal/imap/tls.go` + test mevcut.                                        |
| T030   | IMAP testleri                          | Tamamlandı    | 17 test dosyası — en kapsamlı test coverage.                                 |

**F004 Durumu: TAMAMLANDI**

---

## F005: REST API & Authentication

| Task   | Açıklama                                | Durum         | Detay                                                                         |
|--------|----------------------------------------|---------------|-------------------------------------------------------------------------------|
| T031   | Echo router ve middleware              | Tamamlandı    | `internal/api/router.go`, `server.go`, tüm middleware'ler mevcut.             |
| T032   | JWT authentication                     | Tamamlandı    | `middleware/auth.go`, `service/auth.go` + testler.                            |
| T033   | User CRUD endpoints                    | Tamamlandı    | `handlers/users.go` — 13 metot.                                              |
| T034   | Message endpoints                      | Tamamlandı    | `handlers/messages.go` — 25+ metot (list, get, delete, mark, move, bulk).    |
| T035   | Attachment endpoints                   | Tamamlandı    | `handlers/attachments.go` — list, download mevcut.                            |
| T036   | Search endpoints                       | Tamamlandı    | `handlers/search.go` — simple ve advanced search.                             |
| T037   | Webhook endpoints                      | Tamamlandı    | `handlers/webhooks.go` — CRUD + activate/deactivate/test.                     |
| T038   | Mailbox endpoints                      | Tamamlandı    | `handlers/mailboxes.go` — CRUD + stats.                                      |
| T039   | Rate limiting middleware               | Tamamlandı    | `middleware/ratelimit.go` + test.                                             |
| T040   | CORS middleware                        | Tamamlandı    | `middleware/cors.go` + test.                                                  |
| T041   | Security headers                       | Tamamlandı    | `middleware/security.go` + test.                                              |
| T042   | Swagger/OpenAPI                        | EKSİK         | Config'de `EnableSwagger` var ama endpoint/handler yok.                       |

**F005 Durumu: %95 TAMAMLANDI** — Swagger endpoint eksik.

---

## F006: Web UI

| Task   | Açıklama                                | Durum         | Detay                                                                         |
|--------|----------------------------------------|---------------|-------------------------------------------------------------------------------|
| T043   | SvelteKit proje yapısı                 | Tamamlandı    | `web/` — SvelteKit 2 + Svelte 5 + Tailwind.                                  |
| T044   | Login sayfası                          | Tamamlandı    | `web/src/routes/login/+page.svelte`.                                          |
| T045   | Message detail view                    | Tamamlandı    | `web/src/routes/message/[id]/+page.svelte`.                                   |
| T046   | Inbox view                             | Tamamlandı    | `web/src/routes/inbox/+page.svelte`.                                          |
| T047   | Settings ve webhook sayfaları          | Tamamlandı    | `web/src/routes/settings/+page.svelte`.                                       |
| T048   | Dashboard                              | Tamamlandı    | `web/src/routes/+page.svelte` (dashboard).                                    |
| T049   | User management                        | Tamamlandı    | `web/src/routes/users/+page.svelte`.                                          |

**F006 Durumu: TAMAMLANDI**

---

## F007: Multi-Database Support

| Task   | Açıklama                                | Durum         | Detay                                                                         |
|--------|----------------------------------------|---------------|-------------------------------------------------------------------------------|
| T050   | PostgreSQL repository                  | Tamamlandı    | `internal/repository/postgres/` — 11 dosya.                                   |
| T051   | MySQL repository                       | Tamamlandı    | `internal/repository/mysql/` — 10 dosya.                                      |
| T052   | MongoDB repository                     | Tamamlandı    | `internal/repository/mongodb/` — 12 dosya.                                    |
| T053   | FTS PostgreSQL                         | Tamamlandı    | `postgres/migrations/003_full_text_search.sql`.                               |
| T054   | FTS MySQL                              | Tamamlandı    | `mysql/migrations/003_full_text_search.sql`.                                  |
| T055   | FTS SQLite                             | Tamamlandı    | `sqlite/migrations/003_full_text_search.sql` (bu oturumda eklendi).           |
| T056   | MongoDB indexes                        | Tamamlandı    | `mongodb/indexes.go`.                                                         |
| T057   | Repository factory                     | Tamamlandı    | `internal/repository/factory/factory.go` + test.                              |

**F007 Durumu: TAMAMLANDI**

---

## F008: Docker & Deployment

| Task   | Açıklama                                | Durum         | Detay                                                                         |
|--------|----------------------------------------|---------------|-------------------------------------------------------------------------------|
| T058   | Dockerfile                             | Tamamlandı    | Multi-stage Dockerfile mevcut.                                                |
| T059   | docker-compose.yml                     | Tamamlandı    | SQLite, PostgreSQL, MySQL, MongoDB profilleri.                                |
| T060   | CI/CD — Docker workflow                | Tamamlandı    | `.github/workflows/docker.yml`.                                               |
| T061   | CI/CD — Release workflow               | Tamamlandı    | `.github/workflows/release.yaml` (GoReleaser).                               |
| T062   | CI/CD — Test/lint workflow             | Tamamlandı    | `.github/workflows/ci.yml` (bu oturumda eklendi).                             |
| T063   | Kubernetes manifests                   | Tamamlandı    | `deployments/kubernetes/` — deployment, service, ingress, pvc, configmap.     |
| T064   | Reverse proxy configs                  | Tamamlandı    | `examples/nginx/`, `examples/traefik/`.                                       |
| T065   | Production docs                        | Tamamlandı    | `docs/deployment.md`, `production.md`, `reverse-proxy.md`, `backup-restore.md`. |

**F008 Durumu: TAMAMLANDI**

---

## CLI Komutları — TODO'lar

| Dosya              | TODO Sayısı | Detay                                                                              |
|--------------------|-------------|------------------------------------------------------------------------------------|
| `cmd_messages.go`  | 6           | list, view, delete, purge, export, stats — hepsi placeholder.                     |
| `cmd_migrate.go`   | 4           | migrate up, down, status, reset — hepsi placeholder.                              |
| `cmd_user.go`      | 6           | list, create, delete, password, info, activate/deactivate — hepsi placeholder.    |
| `cmd_health.go`    | 1           | Database connectivity check placeholder.                                          |

**Toplam: 17 TODO** — CLI komutları repository/service bağlantısı yapılmamış.

---

## Test Durumu

| Paket                        | Test Dosyası | Durum                                                          |
|------------------------------|-------------|----------------------------------------------------------------|
| `internal/domain/`           | 10          | Kapsamlı.                                                      |
| `internal/imap/`             | 17          | En kapsamlı coverage.                                          |
| `internal/smtp/`             | 7           | Mevcut ama flaky rate limit testi var.                         |
| `internal/service/`          | 7           | auth, mailbox, message, relay + user, webhook, notify testleri.|
| `internal/api/handlers/`     | 3           | auth, health, users testleri. Eksik: messages, mailboxes, webhooks, search, attachments, system. |
| `internal/api/middleware/`   | 6           | Kapsamlı.                                                      |
| `internal/api/`              | 3           | router, server, response testleri.                             |
| `internal/config/`           | 3           | config, logger, validation testleri.                           |
| `internal/repository/`       | 6           | sqlite, factory, integration, benchmark, helpers.              |
| `internal/parser/`           | 1           | mime_test.go.                                                  |

---

## Özet — Kategorilere Göre Eksikler

### Kritik (P0)

| #  | Eksiklik                                                | Etki                                                   |
|----|--------------------------------------------------------|-------------------------------------------------------|
| 1  | CLI komutları placeholder (17 TODO)                    | `yunt user`, `yunt messages`, `yunt migrate` çalışmıyor. |

### Yüksek (P1)

| #  | Eksiklik                                                | Etki                                                   |
|----|--------------------------------------------------------|-------------------------------------------------------|
| 2  | Handler testleri eksik (6/10 handler)                  | messages, mailboxes, webhooks, search, attachments, system handler'ları test edilmemiş. |
| 3  | Swagger/OpenAPI endpoint yok                           | API dokümantasyonu sunulmuyor.                          |
| 4  | SQLite repository test hataları                        | Pagination off-by-one, stats scan hataları.             |

### Orta (P2)

| #  | Eksiklik                                                | Etki                                                   |
|----|--------------------------------------------------------|-------------------------------------------------------|
| 5  | E2E testler yok                                        | HTTP API, SMTP, IMAP uçtan uca test edilmemiş.          |
| 6  | SMTP rate limit testi flaky                            | CI'da random fail edebilir.                             |
| 7  | WebSocket real-time updates yok                        | Web UI polling kullanıyor, real-time değil.             |

### Düşük (P3)

| #  | Eksiklik                                                | Etki                                                   |
|----|--------------------------------------------------------|-------------------------------------------------------|
| 8  | `CGO_ENABLED=0` goreleaser'da                          | Release binary'leri SQLite içermiyor.                   |
| 9  | Webhook dispatch doğrulaması                           | HTTP POST, HMAC signing, retry mekanizması doğrulanmalı.|
