# Changelog

Alle nennenswerten Änderungen dieses Projekts werden in dieser Datei
dokumentiert. Das Format folgt [Keep a Changelog](https://keepachangelog.com/de/1.1.0/),
und das Projekt verwendet [Semantic Versioning](https://semver.org/lang/de/).

## [1.0.0] - 2026-09-03

### Hinzugefügt

- Pastebin-REST-API in Go (nur Standardbibliothek `net/http`), ohne externe
  Abhängigkeiten.
- `POST /pastes` zum Anlegen eines Pastes mit optionaler Ablaufzeit
  (`expires_in_seconds`) und optionaler Sprache.
- `GET /pastes/{id}` zum Abrufen eines einzelnen Pastes über eine
  kryptographisch zufällige, 128-Bit-ID (Capability-Modell).
- `GET /pastes` zum Auflisten der Metadaten, abgesichert per API-Key
  (`X-API-Key` / `PASTEBIN_API_KEY`).
- `DELETE /pastes/{id}` zum Löschen, abgesichert per Delete-Token
  (`X-Delete-Token`).
- `GET /health` als Health-Check.
- Einheitliche JSON-Fehlerantworten (Status ≥ 400) ohne interne Details.
- Thread-sicherer In-Memory-Store (`sync.Mutex`) mit automatischem
  Ablauf-Cleanup (Hintergrund-Goroutine) und Begrenzung auf 10 000 Pastes.
- Begrenzung des Request-Bodys auf 1 MB (`413` bei Überschreitung).

### Geändert

- Transport- und CORS-Härtung: optionale TLS-Terminierung im Prozess
  (`CERT_FILE` / `KEY_FILE`) mit HSTS, und CORS-Allowlist über
  `CORS_ALLOWED_ORIGINS` (Standard: alle Origins verweigert).

### Sicherheit

- Zugriffsschutz über Ownership-Token (Delete-Token) und API-Key-Autorisierung
  für die Auflistung.
- `crypto/rand` für ID- und Token-Erzeugung (128 Bit Entropie);
  konstantzeitiger Token-Vergleich (`crypto/subtle`).
