# Pastebin API

Eine kleine Pastebin-REST-API in Go, ausschließlich mit der Standardbibliothek
(`net/http`) — ohne externe Web-Frameworks. Pastes werden thread-safe in einem
In-Memory-Store gehalten und können optional nach einer angegebenen
Sekundenzahl ablaufen. Die API unterstützt Anlegen, Abrufen, Auflisten und
Löschen mit sauberen Statuscodes und einheitlichen JSON-Fehlern.

## Tech Stack

- **Sprache:** Go
- **Framework:** `net/http` (Standardbibliothek)
- **Build:** `go.mod` (Modul `pastebin`)
- **Tests:** `go test` / `net/http/httptest`

## Installation

Voraussetzung: Go 1.22 oder neuer (die Route-Muster mit `{id}` benötigen die
neue `ServeMux`-Syntax).

```sh
git clone <repo-url>
cd <repo>
```

Keine externen Abhängigkeiten — `go mod tidy` ist nicht nötig.

## Ausführen

```sh
go run .
```

Der Server lauscht danach auf Port `8080`.

## Endpunkte

Jede JSON-Antwort setzt `Content-Type: application/json; charset=utf-8` und
den CORS-Header `Access-Control-Allow-Origin: *`.

### `GET /health`

Health-Check.

- **Antwort:** `200`

```json
{ "status": "ok" }
```

### `POST /pastes`

Legt einen neuen Paste an.

- **Body:** `{"content": "str" (pflicht), "language": "str" (optional), "expires_in_seconds": "int" (optional, > 0)}`
- **Antwort:** `201` mit `{"id": "<32 Hex-Zeichen>", "delete_token": "<32 Hex-Zeichen>"}`
- `400` bei leerem/fehlendem `content` oder ungültigem JSON
- `413` bei einem Body größer als 1 MB

Das zurückgegebene `delete_token` autorisiert die spätere Löschung des Pastes
(siehe [Authentifizierung und Zugriffsschutz](#authentifizierung-und-zugriffsschutz)).

### `GET /pastes/{id}`

Ruft einen einzelnen Paste ab.

- **Antwort:** `200` mit `{"id", "content", "language", "created_at" (RFC3339), "expires_at"? (RFC3339, nur bei Ablauf)}`
- `404` bei unbekannter ID

### `GET /pastes`

Listet alle Pastes ohne Inhalt. Erfordert einen gültigen API-Key im Header
`X-API-Key` (siehe [Authentifizierung und Zugriffsschutz](#authentifizierung-und-zugriffsschutz)).

- **Antwort:** `200` mit `[{"id", "language", "created_at", "expires_at"?}]`
- `401` bei fehlendem oder ungültigem API-Key

### `DELETE /pastes/{id}`

Löscht einen Paste. Erfordert das beim Anlegen zurückgegebene Lösch-Token im
Header `X-Delete-Token`.

- **Antwort:** `204` ohne Body
- `401` bei fehlendem oder ungültigem Lösch-Token
- `404` bei unbekannter ID

### Fehler

Alle Antworten mit Status ≥ 400 sind ein JSON-Objekt mit ausschließlich einem
generischen `error`-Feld:

```json
{ "error": "not found" }
```

Eine falsche HTTP-Methode auf einem bekannten Pfad antwortet mit `405`, einem
`Allow`-Header und einem JSON-Fehler. Unbekannte Pfade antworten mit `404`.

## Authentifizierung und Zugriffsschutz

Der Dienst kennt zwei getrennte Autorisierungsmechanismen:

- **Lösch-Token (`delete_token`):** Beim Anlegen eines Pastes über
  `POST /pastes` liefert die Antwort ein kryptographisch zufälliges
  `delete_token` zurück. Nur der Ersteller erhält dieses Token. Die Löschung
  erfolgt über `DELETE /pastes/{id}` und erfordert das Token im Header
  `X-Delete-Token`. Ohne gültiges Token wird die Löschung mit `401` verweigert.
- **API-Key (`X-API-Key`):** Die Auflistung `GET /pastes` ist geschützt und
  erfordert einen API-Key im Header `X-API-Key`, der mit der Umgebungsvariable
  `PASTEBIN_API_KEY` übereinstimmen muss. Ist kein Key konfiguriert, antwortet
  der Endpunkt mit `401`.

Der Zugriffsschutz ist im Einzelnen:

| Endpunkt          | Autorisierung                                  |
| ----------------- | ---------------------------------------------- |
| `POST /pastes`    | keine; liefert `delete_token`                  |
| `GET /pastes/{id}`| keine; öffentlich über die zufällige ID        |
| `GET /pastes`     | `X-API-Key` (gleich `PASTEBIN_API_KEY`)        |
| `DELETE /pastes/{id}` | `X-Delete-Token` (gleich `delete_token`)  |

## Umgebungsvariablen

| Variable                | Pflicht | Beschreibung                                                                 |
| ----------------------- | ------- | ---------------------------------------------------------------------------- |
| `PORT`                  | nein    | Port, auf dem der Server lauscht (Standard `8080`).                          |
| `CERT_FILE`             | nein    | Pfad zum TLS-Server-Zertifikat (PEM). Gemeinsam mit `KEY_FILE` aktiviert TLS. |
| `KEY_FILE`              | nein    | Pfad zum privaten TLS-Schlüssel (PEM). Gemeinsam mit `CERT_FILE` aktiviert TLS. |
| `CORS_ALLOWED_ORIGINS`  | nein    | Kommagetrennte Liste erlaubter Ursprünge für CORS. Leer = alle verweigert.    |
| `PASTEBIN_API_KEY`      | nein    | API-Key für `GET /pastes`. Leer = Auflistung deaktiviert (`401`).             |

## Datenschutz und Sicherheit

- `PRIVACY.md` — Datenschutzerklärung: Zweck, Rechtsgrundlage, Speicherdauer,
  Löschkonzept, Betroffenenrechte und verantwortliche Kontaktstelle.
- `SECURITY.md` — Sicherheitsbewertung und Härtungsmaßnahmen.

## Features

- Thread-sicherer In-Memory-Store (`sync.Mutex`)
- Optionale Ablaufzeit für Pastes; abgelaufene Pastes werden beim Zugriff
  entfernt
- Einheitliche JSON-Fehler ohne interne Details
- CORS (`Access-Control-Allow-Origin: *`) auf allen Antworten
