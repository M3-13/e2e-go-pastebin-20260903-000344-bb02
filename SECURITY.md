# Sicherheitsrichtlinie

Dieses Dokument beschreibt das Sicherheitsmodell der Pastebin-REST-API, die
unterstützten Versionen, den Meldeweg für Sicherheitslücken und die
Update-/Patch-Prozedur. Es richtet sich an Betreiber, Entwickler und
Sicherheitsforscher.

## Sicherheitsmodell

Die API ist ein öffentlicher Pastebin-Dienst. Es gibt keine
Benutzerkonten und keine serverseitige Sitzung. Der Zugriffsschutz beruht
stattdessen auf drei voneinander unabhängigen Mechanismen:

### 1. Linkbasiertes Lesen über eine unknackbare ID

Jeder Paste erhält beim Anlegen eine zufällige ID aus 32 Hex-Zeichen, die aus
16 Bytes von `crypto/rand` erzeugt wird (128 Bit Entropie). Die ID ist der
einzige Schlüssel zum Lesen: `GET /pastes/{id}` ist unauthentifiziert und
liefert den Inhalt an jeden, der die ID kennt (Capability-Modell).

- Wer die ID nicht kennt, kann den Paste nicht abrufen; ein Erraten ist
  kryptographisch nicht praktikabel.
- Die ID wird in der URL übertragen und gilt als vertraulich. Sie darf nicht
  protokolliert, in Referrer-Headern oder in Zugriffsprotokollen Dritter
  auftauchen (der Betreiber muss dies in vorgelagerten Proxies unterbinden).

### 2. Delete-Token für das Löschen

Beim Anlegen liefert `POST /pastes` zusätzlich zur ID ein `delete_token`
(ebenfalls 32 Hex-Zeichen aus `crypto/rand`). Löschen ist nur mit diesem Token
möglich: `DELETE /pastes/{id}` verlangt den Header `X-Delete-Token`, der dem
Token des Pastes entsprechen muss. Der Vergleich erfolgt in konstanter Zeit
(`crypto/subtle.ConstantTimeCompare`), um Timing-Angriffe auszuschließen.

- Ohne gültiges Token antwortet die API mit `401`.
- Das Token wird ausschließlich in der Antwort auf `POST /pastes` zurückgegeben
  und nirgends gespeichert oder protokolliert. Es ist geheim zu behandeln.
- Wer ein `delete_token` verliert, kann den zugehörigen Paste nicht löschen.

### 3. API-Key für die Auflistung

Die Metadatenauflistung `GET /pastes` ist kein öffentlicher Endpunkt: Sie
verlangt den Header `X-API-Key`, der dem in der Umgebungsvariable
`PASTEBIN_API_KEY` hinterlegten Wert entsprechen muss. Ist die Variable nicht
gesetzt oder leer, ist der Endpunkt deaktiviert und antwortet immer mit `401`.

- Der API-Key wird ausschließlich über die Umgebung konfiguriert und darf nie
  im Repository oder in der Dokumentation als Klartextwert stehen.
- Es gibt keine Unterscheidung nach Mandant oder Nutzer; ein gültiger Key
  gewährt Zugriff auf die gesamte Liste.

### Weitere Schutzmaßnahmen

- **Begrenzung des Request-Bodys:** `POST /pastes` liest höchstens 1 MB
  (`http.MaxBytesReader`); größere Anfragen werden mit `413` abgewiesen.
- **Begrenzung der Ablaufzeit:** `expires_in_seconds` ist auf etwa 10 Jahre
  begrenzt; größere Werte werden mit `400` abgewiesen, um einen
  `time.Duration`-Überlauf auszuschließen.
- **Begrenzung der Pastes-Anzahl:** Der Store fasst höchstens 10 000 Pastes
  (`store.MaxPastes`). Darüber hinaus schlägt das Anlegen mit `503` fehl.
- **Automatischer Ablauf-Cleanup:** Abgelaufene Pastes werden beim Zugriff
  (`Get`/`List`) sowie durch eine Hintergrund-Goroutine entfernt.
- **Einheitliche Fehlerantworten:** Alle Antworten mit Status ≥ 400 sind ein
  JSON-Objekt mit ausschließlich einem generischen `error`-Feld — ohne
  Stacktraces, Dateipfade oder interne Details.
- **Transportverschlüsselung:** Der Server unterstützt TLS über die
  Umgebungsvariablen `CERT_FILE` und `KEY_FILE`. Sind beide gesetzt, wird
  `ListenAndServeTLS` verwendet und zusätzlich der HSTS-Header
  `Strict-Transport-Security` gesetzt. Ohne eigene Zertifikate ist der Server
  für den Betrieb hinter einem TLS-terminierenden Reverse-Proxy vorgesehen und
  darf nicht direkt ins Internet exponiert werden.
- **CORS-Allowlist:** Der `Access-Control-Allow-Origin`-Header wird nur für
  Origins gesetzt, die in `CORS_ALLOWED_ORIGINS` (kommagetrennt) aufgeführt
  sind. Eine leere Allowlist setzt keinen Header und verweigert damit
  standardmäßig jeden Cross-Origin-Zugriff.

## Unterstützte Versionen

Es wird ausschließlich die jeweils aktuelle Version unterstützt. Ältere
Versionen erhalten keine Sicherheits-Patches.

| Version | Unterstützt          |
| ------- | -------------------- |
| 1.0.0   | :white_check_mark:   |

## Meldung von Sicherheitslücken

Wenn du eine Sicherheitslücke in diesem Projekt entdeckst, melde sie bitte
**nicht** über ein öffentliches Issue, sondern vertraulich an den Betreiber:

- **E-Mail:** `security@example.invalid` (bitte durch die tatsächliche
  Betreiberadresse ersetzen)
- **Betreff:** `[Sicherheitslücke] <Kurzbeschreibung>`

Bitte gib in der Meldung so viel Kontext wie möglich an:

- betroffene Version und Komponente,
- eine Beschreibung der Schwachstelle und ihres Auswirkungen,
- Schritte zur Reproduktion (sofern vorhanden),
- ein Proof-of-Concept oder eine betroffene URL, falls möglich.

Wir bestätigen den Eingang innerhalb von 5 Werktagen und streben an, innerhalb
von 90 Tagen eine Lösung zu veröffentlichen. Wir veröffentlichen Details erst,
nachdem ein Fix bereitsteht. Sobald die Schwachstelle behoben ist, wird sie im
CHANGELOG und ggf. in einer Sicherheitsadvisory dokumentiert.

## Update- und Patch-Prozedur

1. Sicherheitsrelevante Änderungen werden als regulärer Fix auf `main` gemergt
   und über den CI-Pfad gebaut und getestet (`go build ./...`, `go test ./...`).
2. Jede Änderung wird im [CHANGELOG.md](CHANGELOG.md) unter der entsprechenden
   Version dokumentiert.
3. Ein Update besteht aus dem Einspielen der neuen Version und dem Neustart des
   Dienstes über den in `RUN.json` deklarierten Startbefehl (`go run .`).
4. Nach einem Update sind die Umgebungsvariablen (`CERT_FILE`, `KEY_FILE`,
   `CORS_ALLOWED_ORIGINS`, `PASTEBIN_API_KEY`) zu prüfen und, wo nötig, zu
   erneuern (insbesondere API-Keys und Zertifikate rotieren).
5. Wird eine Schwachstelle in einer Drittanbieter-Abhängigkeit gemeldet, ist
   die betroffene Version zu ermitteln und ein Upgrade einzuspielen. Aktuell
   verwendet das Projekt keine externen Abhängigkeiten über die
   Go-Standardbibliothek hinaus (siehe `sbom.json`).
