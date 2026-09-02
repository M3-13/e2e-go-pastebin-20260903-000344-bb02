VERDICT: CHANGES_REQUESTED

Manuelle Analyse des Go-Backends. Es wurde kein Scanner-Output für diesen Projekttyp geliefert; die Bewertung basiert auf der Codeanalyse. Es wurden keine kritischen oder hohen Schwachstellen gefunden, aber mehrere mittlere und niedrige Befunde.

## Sicherheitsbericht

### 1. Fehlende TLS-Erzwingung im Produktivmodus
- **Schweregrad:** mittel
- **Datei/Stelle:** `main.go`, `main()` und `newApp()`
- **Problem:** Wenn `CERT_FILE`/`KEY_FILE` nicht gesetzt sind, startet der Server unverschlüsselt über `http.ListenAndServe`. Dadurch werden der `X-Delete-Token` und der `X-API-Key` im Klartext übertragen und können auf dem Netzwerkweg mitgelesen werden. Der Server unterstützt TLS nur, wenn beide Dateien konfiguriert sind; ein unverschlüsselter Produktivbetrieb ist ohne Zwang möglich.
- **Konkrete Lösung:** Im Produktivmodus erzwingen, dass TLS konfiguriert ist oder dass ausschließlich ein vorgeschalteter TLS-Terminator verwendet wird. Beispielsweise bei leerem `CERT_FILE`/`KEY_FILE` mit einem expliziten Fehler abbrechen, sofern nicht ausdrücklich ein Development-Modus gesetzt ist. Alternativ abgesicherten Betrieb hinter einem TLS-Reverse-Proxy verbindlich machen. Die bestehende HSTS-Middleware sollte weiterhin nur im TLS-Modus aktiv sein.

### 2. Kein Rate-Limiting; Speicher-Erschöpfung möglich
- **Schweregrad:** mittel
- **Datei/Stelle:** `handler/create.go`, `store/store.go`, `main.go`
- **Problem:** `POST /pastes` akzeptiert unbegrenzte Anfragen ohne Rate-Limiting. `MaxPastes` liegt bei 10.000 Pastes, und jeder Paste kann bis zu 1 MB Content haben. Ein Angreifer kann den In-Memory-Store mit hohem Speicherverbrauch füllen (theoretisch bis zu ~10 GiB) und damit die Verfügbarkeit des Dienstes beeinträchtigen; anschließend erhalten Neuanlagen 503.
- **Konkrete Lösung:** Rate-Limiting pro Client-IP oder API-Key als Middleware einführen. Zusätzlich ein globales Speicher-Budget (Bytes) durchsetzen und/oder `MaxPastes` sowie `maxPasteBodyBytes` konservativer bzw. konfigurierbar gestalten. Bei Überschreitung sauber 429 oder, falls der Store voll ist, einen klar dokumentierten 503 zurückgeben.

### 3. X-API-Key-Vergleich nicht zeitkonstant
- **Schweregrad:** niedrig
- **Datei/Stelle:** `handler/list.go`, Funktion `apiKeyAuthorized`
- **Problem:** Der API-Key wird mit `r.Header.Get("X-API-Key") == expected` verglichen. Dieser Vergleich läuft nicht in konstanter Zeit und kann theoretisch Timing-Angriffe auf den List-API-Key erleichtern.
- **Konkrete Lösung:** `crypto/subtle.ConstantTimeCompare([]byte(got), []byte(expected)) == 1` verwenden, analog zum Delete-Token-Vergleich in `handler/delete.go`.

### 4. Fehlende Cache-Control-Header
- **Schweregrad:** niedrig
- **Datei/Stelle:** `handler/handler.go`, `WriteJSON`
- **Problem:** Alle JSON-Antworten werden ohne `Cache-Control` ausgeliefert. Besonders kritisch ist die Antwort von `POST /pastes`, die den `delete_token` enthält, sowie `GET /pastes/{id}`, der den eigentlich vertraulichen Paste-Inhalt liefert. Browser oder Zwischen-Proxys können diese Antworten cachen.
- **Konkrete Lösung:** In `WriteJSON` vor dem Schreiben des Bodys `Cache-Control: no-store` setzen. Das verhindert, dass sensible Inhalte und Delete-Tokens in Browser- oder Proxy-Caches verbleiben, ohne die API-Funktion zu beeinträchtigen.

### 5. JSON-Decoder akzeptiert anhängende Daten
- **Schweregrad:** niedrig
- **Datei/Stelle:** `handler/create.go`, `CreatePaste`
- **Problem:** `json.NewDecoder(r.Body).Decode(&req)` liest nur das erste JSON-Objekt und ignoriert weitere Daten im Request-Body. Ein Body wie `{"content":"x"}{"evil":true}` wird akzeptiert, obwohl er formal mehr als ein JSON-Objekt enthält. Das ist derzeit nicht direkt ausnutzbar, aber eine unsaubere Eingabevalidierung.
- **Konkrete Lösung:** Body nach dem initialen Decode vollständig auf nur Whitespace prüfen oder alternativ den Body mit `io.ReadAll` im Rahmen des Limits lesen und anschließend mit `json.Unmarshal` verarbeiten. `http.MaxBytesReader` weiterhin zuerst setzen.

### 6. CORS-Preflight nicht vollständig behandelt
- **Schweregrad:** niedrig
- **Datei/Stelle:** `main.go`, `corsMiddleware`
- **Problem:** Die Middleware setzt bei erlaubtem Origin nur `Access-Control-Allow-Origin`. Browser-Sendeanfragen mit `Content-Type: application/json`, `X-Delete-Token` oder `X-API-Key` lösen einen Preflight (`OPTIONS`) aus. Dieser wird aktuell als `405 Method Not Allowed` beantwortet, ohne die nötigen CORS-Header. Dadurch können legitime Browser-Clients mit erlaubtem Origin die API nicht nutzen; die erlaubte Ursprungseinschränkung selbst ist korrekt.
- **Konkrete Lösung:** `OPTIONS`-Preflight-Behandlung ergänzen, die für erlaubte Origins `Access-Control-Allow-Methods: GET, POST, DELETE` und `Access-Control-Allow-Headers: Content-Type, X-Delete-Token, X-API-Key` setzt, sowie `Vary` entsprechend erweitert. So bleibt die Allowlist streng, die API wird aber für die eigenen legitimen Browser-Clients nutzbar.

## Positiv geprüft
- Keine hartkodierten Secrets; API-Key und Zertifikatspfade kommen aus der Umgebung.
- Paste-IDs und Delete-Token werden mit `crypto/rand` erzeugt (128 Bit Entropie).
- Delete-Token-Vergleich in `handler/delete.go` nutzt bereits `crypto/subtle.ConstantTimeCompare`.
- Fehlerantworten sind generisch und enthalten keine internen Details/Stacktraces.
- `model.Paste` serialisiert den Delete-Token nicht (`json:"-"`); `GET /pastes` liefert nur Metadaten ohne Inhalt.
- Abgelaufene Pastes werden bei Zugriff und periodisch entfernt.
- Standard-CORS-Verhalten ist deny-by-default; kein Wildcard `*`.