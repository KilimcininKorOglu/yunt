# Yunt Eksiklik Raporu

Task dosyaları ile mevcut kod tabanının karşılaştırmalı analizi.

## Kritik Eksikler

| #  | Eksiklik                                    | Durum         | Detay                                                                                         |
|----|---------------------------------------------|---------------|-----------------------------------------------------------------------------------------------|
| 1  | LICENSE dosyası yok                         | TAMAMLANDI    | `420f526` — MIT LICENSE eklendi.                                                              |
| 2  | Graceful shutdown tamamlanmamış             | TAMAMLANDI    | `345d038` — Gerçek server başlatma ve graceful shutdown eklendi.                               |
| 3  | WebSocket gerçek implementasyonu yok        | TAMAMLANDI    | `5d9700b` — Kullanılmayan bağımlılık kaldırıldı. Polling ile devam. WebSocket ileride eklenebilir. |
| 4  | Swagger/OpenAPI endpoint'i yok              | Eksik         | Config'de `EnableSwagger` flag'i var ama gerçek Swagger handler/route yok.                    |
| 5  | SQLite FTS5 migration eksik                 | TAMAMLANDI    | `0d72e20` — FTS5 migration eklendi.                                                           |
| 6  | Gerçek server başlatma ve bağlantı yok      | TAMAMLANDI    | `345d038` — Repository, service, handler wiring tamamlandı.                                   |

## Test Eksikleri

| #  | Eksiklik                                    | Durum         | Detay                                                                                         |
|----|---------------------------------------------|---------------|-----------------------------------------------------------------------------------------------|
| 7  | `internal/service/user.go` testi yok        | DEVAM EDİYOR  | Test yazılıyor.                                                                               |
| 8  | `internal/service/webhook.go` testi yok     | DEVAM EDİYOR  | Test yazılıyor.                                                                               |
| 9  | `internal/service/notify.go` testi yok      | DEVAM EDİYOR  | Test yazılıyor.                                                                               |
| 10 | Handler testleri yetersiz                   | Yarım         | Sadece `auth_test.go` var. Diğer handler'lar için test yok.                                   |
| 11 | E2E / entegrasyon testleri yetersiz         | Yarım         | Sadece `repository/integration_test.go` var.                                                  |

## Fonksiyonel Eksikler

| #  | Eksiklik                                    | Durum         | Detay                                                                                         |
|----|---------------------------------------------|---------------|-----------------------------------------------------------------------------------------------|
| 12 | Webhook dispatch/delivery mekanizması       | Belirsiz      | `service/webhook.go` (24KB) mevcut ama HTTP dispatch doğrulanmalı.                            |
| 13 | SMTP TLS sertifika yükleme                  | Mevcut        | STARTTLS yapılandırması ve TLS state tracking var.                                            |
| 14 | SMTP Relay                                  | Mevcut        | `service/relay.go`, `smtp/session.go:relayMessage()` mevcut.                                  |
| 15 | Refresh Token                               | Mevcut        | Implementasyon ve testler mevcut.                                                             |
| 16 | Pagination                                  | Mevcut        | Cursor-based ve offset-based pagination tanımlı.                                              |
| 17 | RBAC                                        | Mevcut        | `middleware/rbac.go` mevcut.                                                                  |
| 18 | Rate Limiting                               | Mevcut        | `middleware/ratelimit.go` + testler mevcut.                                                   |
| 19 | Full-text Search (tüm DB'ler)               | TAMAMLANDI    | SQLite FTS5 migration eklendi. PostgreSQL ve MySQL zaten mevcuttu.                            |

## Altyapı / DevOps Eksikleri

| #  | Eksiklik                                    | Durum         | Detay                                                                                         |
|----|---------------------------------------------|---------------|-----------------------------------------------------------------------------------------------|
| 20 | CI test pipeline yok                        | TAMAMLANDI    | `27cdbe8` — CI workflow eklendi (test + lint + vet).                                          |
| 21 | `.gitkeep` dosyaları temizlenmemiş          | TAMAMLANDI    | `a84069f` — Stale `.gitkeep` dosyaları silindi.                                               |
| 22 | Release workflow çakışması                  | TAMAMLANDI    | `b403442` — Duplicate `release.yml` silindi. GoReleaser kalıyor.                              |

## Ek Düzeltmeler

| #  | Düzeltme                                    | Commit        | Detay                                                                                         |
|----|---------------------------------------------|---------------|-----------------------------------------------------------------------------------------------|
| A1 | `cmd_health.go` IPv6 uyumluluğu             | `daaaf42`     | `fmt.Sprintf` yerine `net.JoinHostPort` kullanıldı.                                          |
| A2 | GoReleaser "hermes" referansları             | `4df8444`     | Build id, binary, main path "yunt" olarak güncellendi.                                        |

## Kalan İşler

| Öncelik | Madde                                                    |
|---------|----------------------------------------------------------|
| P1      | Service testleri: user, webhook, notify (devam ediyor)   |
| P1      | Handler testleri (messages, users, mailboxes, vb.)       |
| P2      | Swagger/OpenAPI endpoint implementasyonu                 |
| P2      | E2E test suite'i (HTTP API, SMTP, IMAP)                  |
| P3      | Webhook dispatch mekanizması doğrulaması                 |
