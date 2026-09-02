VERDICT: CHANGES_REQUESTED

## Sicherheitsbewertung

**Hinweis:** Für diesen Projekttyp (`go-backend`) wurden keine Security-Scanner (bandit / pip-audit / npm audit / semgrep) ausgeführt oder geliefert. Die Bewertung beruht daher ausschließlich auf der manuellen Analyse des sichtbaren Quellcodes. Es wurden keine Hinweise auf einkompilierte Secrets, SQL-/Command-Injection, unsichere Deserialisierung, XSS oder SSRF gefunden. Die Kernfunktionen (ID-Erzeugung per `crypto/rand`, Body-Limit, JSON-Fehler, Mutex-Schutz, Entfernen abgelaufener Pastes) sind sauber umgesetzt.

Die folgenden Findings begründen die Entscheidung **CHANGES_REQUESTED** (mittelgradige Härtungsmaßnahmen).

---

### 1. Ressourcenerschöpfung durch unbegrenzte Paste-Anlage (medium)

- **Betroffene Stelle:** `store/store.go` (Store ohne Gesamtlimit), `handler/create.go` (POST `/pastes` ohne Ratenbegrenzung), `main.go` (keine vorgelagerte Begrenzung)
- **Beschreibung:** Jeder Client kann unauthentifiziert beliebig viele Pastes mit jeweils bis zu 1 MB Inhalt anlegen. Da der Store rein im Arbeitsspeicher liegt und weder die Anzahl der Pastes noch die Gesamtspeichermenge begrenzt ist, kann ein Angreifer den Prozess mit vielen großen Pastes füllen und so einen Denial-of-Service (Speicher-/CPU-Ressourcenerschöpfung) auslösen. Es existiert kein Hintergrund-Job, der abgelaufene Einträge entfernt; sie werden nur bei Zugriff über `GET`/`List` gelöscht und können bei ständig neuen Anlagen ungenutzt im Speicher verbleiben.
- **Konkrete Härtung:**
  - Im `Store` eine maximale Anzahl von Pastes oder eine globale Byte-Grenze einführen (z. B. `maxPastes`, `maxTotalBytes`). Beim Überschreiten `Create` ablehnen oder älteste/abgelaufene Einträge verdrängen.
  - Alternativ auf Handler-/Middleware-Ebene ein Rate-Limiting pro Client-IP (z. B. Token-Bucket) für `POST /pastes` ergänzen.
  - Optional einen periodischen Goroutine-Cleanup für abgelaufene Einträge einplanen, damit der Speicher nicht unnötig wächst.

---

### 2. Unverschlüsselter Transport (medium)

- **Betroffene Stelle:** `main.go`, Zeile `http.ListenAndServe(":"+port, newApp())`
- **Beschreibung:** Der Server lauscht ausschließlich über unverschlüsseltes HTTP. Falls die API direkt aus dem Internet erreichbar ist, können Paste-Inhalte, Metadaten und IDs im Klartext mitgelesen werden. Da die Inhalte potenziell sensibel sein können (Pastebin), ist eine Transportverschlüsselung erforderlich.
- **Konkrete Härtung:**
  - Entweder TLS direkt im Go-Prozess aktivieren (z. B. `http.ListenAndServeTLS` mit Server-Zertifikat) oder
  - im Deployment klarstellen, dass der Server nur hinter einem TLS-terminierenden Reverse-Proxy (nginx, Caddy, Load Balancer) betrieben wird und der Go-Port (`8080`) nicht öffentlich exponiert ist.
  - Zusätzlich dokumentieren, dass `PORT` intern und nicht als öffentlicher HTTP-Endpunkt genutzt werden sollte.

---

### 3. Overflow bei `expires_in_seconds` (low)

- **Betroffene Stelle:** `handler/create.go`, Zeile `expires := time.Now().Add(time.Duration(req.ExpiresInSeconds) * time.Second)`
- **Beschreibung:** `req.ExpiresInSeconds` wird als `int` dekodiert und ohne Ober­grenze in `time.Duration` umgewandelt. Bei sehr großen Werten (nahe `math.MaxInt`) kann die Multiplikation mit `time.Second` überlaufen, was zu einem negativen oder unerwartet kleinen `time.Duration` führt. Der Paste läuft dann sofort oder zu einem falschen Zeitpunkt ab. Ein direkter schwerwiegender Exploit ist nicht möglich, aber es handelt sich um eine vermeidbare Robustheitslücke.
- **Konkrete Härtung:**
  - Eine Obergrenze definieren, z. B. `const maxExpireSeconds = 10 * 365 * 24 * 3600` (10 Jahre) oder `math.MaxInt64 / int64(time.Second)`.
  - Vor der Umwandlung prüfen: `if req.ExpiresInSeconds > maxExpireSeconds { WriteError(400, "expires_in_seconds too large"); return }`.
  - Negative Werte werden bereits durch `if req.ExpiresInSeconds > 0` ignoriert – das ist korrekt.

---

### 4. Weit geöffnete CORS-Richtlinie (low / Hinweis)

- **Betroffene Stelle:** `main.go`, `corsMiddleware`
- **Beschreibung:** `Access-Control-Allow-Origin: *` ist gemäß **AC-12** explizit gefordert und für eine API ohne cookies/authentifizierte Sitzungen unkritisch. Sobald die API jedoch um Authentifizierung (z. B. Session-Cookies oder Authorization-Header mit Browser-Zugriff) erweitert wird, würde `*` es beliebigen Websites ermöglichen, Antworten im Browserkontext zu lesen. Aktuell besteht kein unmittelbares Risiko.
- **Konkrete Härtung (für spätere Erweiterungen):**
  - Ursprung konfigurierbar machen (Whitelist) und bei nicht-`*`-Antworten `Vary: Origin` setzen.
  - Preflight-Verhalten (`OPTIONS`) berücksichtigen, sobald browserbasierte Clients mit nicht-einfachen Anfragen verwendet werden.

---

### 5. Eingeschränkte Fehlerbehandlung bei JSON-Dekodierung (Hinweis, kein Finding)

- **Betroffene Stelle:** `handler/create.go`
- **Beschreibung:** `json.NewDecoder(r.Body).Decode(&req)` liest nur das erste JSON-Objekt; nachfolgende Daten werden ignoriert. Das ist bei einem Request-Body unüblich, aber wegen `http.MaxBytesReader` nicht ausnutzbar. Kein Handlungsbedarf, aber bewusst zur Kenntnis genommen.

---

## Zusammenfassung

- **Secrets:** keine gefunden.
- **Injection/Inputs:** keine SQL-/Command-/Path-Injection; Body-Limit und JSON-Validierung vorhanden.
- **AuthN/AuthZ:** keine Authentifizierung implementiert; Zugriffsschutz basiert auf 128-Bit-Zufalls-IDs (Capability-Modell). Für einen öffentlichen Pastebin-Dienst akzeptabel. Kein Auth-Bypass, da keine geschützten Funktionen existieren.
- **Dependencies:** Keine externen Abhängigkeiten erkennbar (`go.mod` enthält nur Moduldefinition und Go-Version); keine bekannten Schwachstellen. Es liefen jedoch keine Dependency-Scanner.
- **Configuration/Transport:** TLS fehlt auf Anwendungsebene (siehe Finding 2). CORS `*` ist spezifikationskonform, aber für spätere Auth-Erweiterungen einzuengen.

**Begründung des Verdikts:** Die gefundenen mittelgradigen Risiken (Ressourcen-DoS, unverschlüsselter Transport) rechtfertigen eine Überarbeitung vor der Auslieferung. Es wurden keine kritischen oder hohen Schwachstellen festgestellt, daher kein `BLOCKED`.