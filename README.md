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
- **Antwort:** `201` mit `{"id": "<32 Hex-Zeichen>"}`
- `400` bei leerem/fehlendem `content` oder ungültigem JSON
- `413` bei einem Body größer als 1 MB

### `GET /pastes/{id}`

Ruft einen einzelnen Paste ab.

- **Antwort:** `200` mit `{"id", "content", "language", "created_at" (RFC3339), "expires_at"? (RFC3339, nur bei Ablauf)}`
- `404` bei unbekannter ID

### `GET /pastes`

Listet alle Pastes ohne Inhalt.

- **Antwort:** `200` mit `[{"id", "language", "created_at", "expires_at"?}]`

### `DELETE /pastes/{id}`

Löscht einen Paste.

- **Antwort:** `204` ohne Body
- `404` bei unbekannter ID

### Fehler

Alle Antworten mit Status ≥ 400 sind ein JSON-Objekt mit ausschließlich einem
generischen `error`-Feld:

```json
{ "error": "not found" }
```

Eine falsche HTTP-Methode auf einem bekannten Pfad antwortet mit `405`, einem
`Allow`-Header und einem JSON-Fehler. Unbekannte Pfade antworten mit `404`.

## Features

- Thread-sicherer In-Memory-Store (`sync.Mutex`)
- Optionale Ablaufzeit für Pastes; abgelaufene Pastes werden beim Zugriff
  entfernt
- Einheitliche JSON-Fehler ohne interne Details
- CORS (`Access-Control-Allow-Origin: *`) auf allen Antworten
