VERDICT: CHANGES_REQUESTED

## Prüfbericht — Pastebin-REST-API in Go (`go-backend`)

Prüfgegenstand ist der im Branch vorhandene, gemergte Produktstand. Reines Backend ohne Endnutzer-UI: Cookie-/Legal-Notice-/Barrierefreiheitspflichten sind daher nicht einschlägig. Geprüft wurden DSGVO, CRA sowie mittelbar Marktreife/Sicherheitsverhalten der API.

Positiv hervorzuheben: keine Logger-Ausgabe von Paste-Inhalten, `crypto/rand` mit 128 Bit für ID/Delete-Token, Größenbegrenzung des Request-Bodys, mutex-geschützter Store, physisches Entfernen abgelaufener Pastes, kein `content`/`delete_token` in `GET /pastes`, JSON-Fehler ohne Stacktraces/Dateipfade sowie HSTS nur im TLS-Modus.

---

## 1. DSGVO / Datenschutz

### DSGVO-D1 — Unbegrenzte Speicherung ohne Ablauf (mittel)
**Datei:** `store/store.go`, `handler/create.go`, `model/paste.go`

Pastes ohne `expires_in_seconds` werden dauerhaft im In-Memory-Store gehalten. Da `content` nach dem Modell beliebige, auch personenbezogene Inhalte enthalten kann, fehlt eine wirksame technische Durchsetzung des Grundsatzes der Speicherbegrenzung nach Art. 5 Abs. 1 lit. e DSGVO.

**Remedy:** Eine betreiberseitig konfigurierbare Höchstspeicherdauer einführen, z. B. `PASTEBIN_MAX_RETENTION_SECONDS`. Wenn gesetzt, muss `CreatePaste` für Pastes ohne eigenes `expires_in_seconds` ein Ablaufdatum auf `now + MaxRetention` setzen. Der Standardwert `0` erhält das aktuelle Verhalten und damit AC-08; im Produktionsbetrieb muss der Betreiber den Wert setzen oder in `PRIVACY.md` eine ausdrücklich begründete Rechtsgrundlage und Löschroutine dokumentieren. Keine pauschale Standard-TTL erzwingen, weil sonst AC-08 bewusst gebrochen würde.

### DSGVO-D2 — Löschweg nur über einmalig ausgegebenen Delete-Token (niedrig)
**Datei:** `handler/delete.go`, `PRIVACY.md`

Eine Löschung ist nur mit dem bei `POST /pastes` zurückgegebenen `X-Delete-Token` möglich. Verliert die betroffene Person den Token, besteht im Produkt kein Löschweg. Das ist kein zwingender Codeblocker, aber betroffen ist die praktische Durchsetzung des Rechts auf Löschung, Art. 17 DSGVO.

**Remedy:** In `PRIVACY.md` einen Betreiberprozess für Löschersuchen dokumentieren (Kontakt, Identifikation der Paste-ID, Fristen). Optional im Code einen administrativen Löschweg hinter dem vorhandenen API-Key-Mechanismus ergänzen, ohne den normalen DELETE-Endpunkt zu verändern.

### DSGVO-D3 — Unverschlüsselter HTTP-Betrieb möglich (hoch)
**Datei:** `main.go`

Die Anwendung fällt auf `http.ListenAndServe` zurück, sobald `CERT_FILE`/`KEY_FILE` nicht gesetzt sind. Paste-Inhalte können personenbezogene Daten enthalten; eine Übertragung ohne TLS verletzt regelmäßig die Anforderungen an die Vertraulichkeit und Integrität nach Art. 32 DSGVO, wenn der Dienst nicht ausschließlich hinter einem TLS-terminierenden Reverse-Proxy betrieben wird.

**Remedy:** Sichere Standardeinstellung herstellen: Ohne `CERT_FILE`/`KEY_FILE` soll der Prozess im Produktionsmodus nicht starten. Für lokale Entwicklung eine ausdrückliche Opt-in-Umgebungsvariable einführen, z. B. `INSECURE_HTTP=1` oder `ALLOW_PLAINTEXT_HTTP=1`. Alternativ in `README.md`/`SECURITY.md` verpflichtend dokumentieren, dass Produktionsbetrieb ausschließlich hinter TLS-Proxy erfolgt; der Code allein sollte den ungesicherten Fall nicht stillschweigend wählen.

### DSGVO-D4 — API-Key-Vergleich nicht in konstanter Zeit (niedrig)
**Datei:** `handler/list.go`

`apiKeyAuthorized` vergleicht den API-Key mit `==`. Das erlaubt potenzielle Timing-Unterschiede und erleichtert Angriffe auf das Geheimnis.

**Remedy:** Analog zu `handler/delete.go` den Vergleich auf `crypto/subtle.ConstantTimeCompare([]byte(expected), []byte(received))` umstellen.

---

## 2. EU Cyber Resilience Act (CRA)

### CRA-C1 — Fehlende sichere Standardeinstellung bei Transportverschlüsselung (hoch)
**Datei:** `main.go`

Identischer Sachverhalt wie DSGVO-D3. Ein Produkt mit digitalen Elementen muss nach CRA mit sicheren Grundeinstellungen ausgeliefert werden. Der automatische Fallback auf Klartext-HTTP widerspricht Security-by-Default.

**Remedy:** Wie DSGVO-D3: TLS erzwingen oder unsicheren HTTP-Betrieb nur über eine explizite Opt-in-Umgebungsvariable erlauben. In `SECURITY.md` die Deployment-Anforderung und den Mechanismus dokumentieren.

### CRA-C2 — CORS-Preflight nicht funktional (mittel)
**Datei:** `main.go`, `handler/handler.go`

`corsMiddleware` setzt lediglich `Access-Control-Allow-Origin` und `Vary`, behandelt aber keine `OPTIONS`-Preflight-Anfragen. Der Mux lässt auf `/pastes` und `/pastes/{id}` nur GET/POST bzw. GET/DELETE zu; `OPTIONS` führt daher zu 405. Damit sind Cross-Origin-Aufrufe mit `Content-Type: application/json`, `X-Delete-Token` oder `X-API-Key` aus Browsern faktisch blockiert. Das ist ein Sicherheits- und Interoperabilitätsdefekt, der auch die dokumentierte CORS-Funktion im Produkt nicht erfüllt.

**Remedy:** In `corsMiddleware` Preflight-Anfragen vor dem Weiterreichen abfangen: Bei `OPTIONS` und vorhandenem `Access-Control-Request-Method` für eine erlaubte Origin direkt mit Status 204 beantworten. Zusätzlich Header setzen:
- `Access-Control-Allow-Methods: GET, POST, DELETE, OPTIONS`
- `Access-Control-Allow-Headers: Content-Type, X-Delete-Token, X-API-Key`
- `Access-Control-Max-Age: 86400`
- `Vary: Origin`

Die normalen, nicht-Preflight-Ressourcen bleiben unverändert; dadurch werden keine vorhandenen API-Aufrufe gebrochen.

### CRA-C3 — Kein Rate-Limit/Schutz vor Ressourcenauslastung (mittel)
**Datei:** `main.go`, `store/store.go`, `handler/create.go`

`MaxPastes` begrenzt die Anzahl, aber bei `MaxPastes = 10000` und je `1 MB` Request-Body kann der Speicher bis in den zweistelligen GB-Bereich getrieben werden. Es fehlt eine Ratenbegrenzung oder IP-/API-Key-bezogene Missbrauchskontrolle. Für Security-by-Default und Verfügbarkeit ist das relevant.

**Remedy:** Middleware zur Ratenbegrenzung einführen, z. B. Token-Bucket pro Quell-IP und für `GET /pastes` pro API-Key. Konfigurierbar über Umgebungsvariablen wie `RATE_LIMIT_REQUESTS_PER_MIN` und `RATE_LIMIT_BURST`, Standardwerte moderat wählen. `MaxPastes` zusätzlich über eine Umgebungsvariable konfigurierbar machen, z. B. `PASTEBIN_MAX_PASTES`.

### CRA-C4 — SBOM und Sicherheitsdokumentation vorhanden, aber nicht inhaltlich prüfbar (niedrig)
**Datei:** `sbom.json`, `SECURITY.md`, `COMPLIANCE.md`, `go.mod`

Die Dateien existieren im Branch, ihr Inhalt ist jedoch nicht Teil des gezeigten Prüfstands. Für CRA-Konformität müssen daraus Abhängigkeiten samt Versionen, unterstützter Zeitraum, Meldeweg für Schwachstellen und Update-/Patch-Prozess hervorgehen.

**Remedy:** Vor Freigabe `sbom.json` gegen `go.mod` validieren. In `SECURITY.md` mindestens folgende Abschnitte sicherstellen: „Security Properties“, „Supported Versions“, „Reporting a Vulnerability“, „Update/Patch Process“. `COMPLIANCE.md` mit den CRA-Bezügen abgleichen. Da die Dateien vorhanden sind, ist dies eine Prüf-/Ergänzungsanforderung, kein Neuaufbau.

---

## 3. EU AI Act

Nicht anwendbar. Im Produkt ist keine KI-Funktion enthalten; es bestehen keine Pflichten nach Risikoklassen, Transparenz- oder Kennzeichnungspflichten.

---

## 4. Pflichttexte & UI

Pflichten für Cookie-Banner, Impressum, Widerrufsbelehrung oder Barrierefreiheit bestehen für dieses reine Backend ohne Endnutzer-Web-UI nicht.

Allerdings ist für den Betrieb als API mit personenbezogenen Inhalten eine Datenschutzinformation für betroffene Personen/API-Kunden erforderlich.

### UI-M1 — Inhalt von `PRIVACY.md` nicht verifizierbar (mittel)
**Datei:** `PRIVACY.md`

Die Datei existiert, ihr Inhalt ist im gezeigten Stand nicht sichtbar. Für DSGVO-Konformität muss sie mindestens abdecken: Verantwortlicher, Zwecke und Rechtsgrundlagen, Kategorien personenbezogener Daten, Speicherdauer/Löschkonzept, Betroffenenrechte, Beschwerderecht bei der Aufsichtsbehörde.

**Remedy:** `PRIVACY.md` inhaltlich gegen Art. 13 und 14 DSGVO prüfen und fehlende Abschnitte ergänzen. Da kein Web-UI existiert, ist kein Cookie- oder Legal-Notice-Banner erforderlich.

---

## 5. Barrierefreiheit / WCAG / BITV / EAA

Nicht anwendbar. Es gibt keine öffentliche Web-Oberfläche. Falls später ein Web-Client bereitgestellt wird, wären die EAA/WCAG-Pflichten für diesen Client neu zu bewerten.

---

## 6. Abstimmung: Auflagen vs. Produktfunktion

Die vorgeschlagenen Maßnahmen sind so gewählt, dass sie bestehende Akzeptanzkriterien nicht brechen:

- TLS-Zwang erfolgt nur mit expliziter Ausnahme für lokale Entwicklung, nicht als kategorische Blockade.
- Die Höchstspeicherdauer wird als optionale, betreiberseitig gesetzte Obergrenze eingeführt; bei `0` bleibt das dauerhafte Abrufen ohne Ablauf gemäß AC-08 erhalten.
- Der CORS-Preflight ergänzt nur die bisher fehlende `OPTIONS`-Behandlung und lässt GET/POST/DELETE unverändert.
- Ratenbegrenzung ist konfigurierbar und mit moderaten Standardwerten umsetzbar.

Damit bestehen keine fundamentalen Rechtsverstöße, die eine Sperrung (`BLOCKED`) rechtfertigen. Die genannten Punkte sind vor Marktfreigabe zu beheben.