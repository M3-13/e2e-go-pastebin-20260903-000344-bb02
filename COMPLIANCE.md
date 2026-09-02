VERDICT: BLOCKED

Geprüft wird der gemergte Stand des `go-backend`-Pastebin-Dienstes. Reines Backend ohne Endnutzer-UI, daher entfallen Cookie-/Legal-Notice-/Accessibility-Pflichten im Code. Geprüft wurden DSGVO und EU Cyber Resilience Act (CRA). Der EU AI Act ist nicht anwendbar, da kein KI-Feature vorhanden ist.

---

## DSGVO

### 1. Kritisch — Öffentlicher, unauthentifizierter Zugriff auf alle Pastes und deren IDs
**Befund:**  
Die API verarbeitet zwingend personenbezogene Daten (Paste-Inhalte sind nutzergeneriert und können personenbezogene Daten enthalten). Es gibt jedoch keinerlei Authentifizierung oder Autorisierung:
- `GET /pastes` listet alle IDs öffentlich auf.
- `GET /pastes/{id}` liefert jeden Paste an jeden, der die ID kennt.
- `DELETE /pastes/{id}` erlaubt jedem, fremde Pastes zu löschen.

Eine Rechtsgrundlage für diese öffentliche Zugänglichmachung ist im Code nicht erkennbar. Es fehlt jede Einwilligung, jedes Vertragsverhältnis oder eine sonstige Rechtsgrundlage nach Art. 6 DSGVO. Dies verletzt Art. 5 Abs. 1 lit. a und f, Art. 25 und Art. 32 DSGVO fundamental.

**Abhilfe:**
- Authentifizierung und Autorisierung einführen, z. B. API-Key pro Mandant/User (`main.go`, `handler/*.go`).
- Beim Anlegen (`POST /pastes`) ein zusätzliches, geheimes Besitz-/Delete-Token zurückgeben, das für `DELETE /pastes/{id}` erforderlich ist (`handler/create.go`, `handler/delete.go`, `model/paste.go`).
- `GET /pastes` nur für berechtigte Nutzer oder ohne öffentliche ID-Auflistung; alternativ die Liste auf eigene Pastes beschränken (`handler/list.go`, `main.go`).
- Diese Maßnahmen sind mit den Funktionen vereinbar, erfordern aber eine Anpassung der Schnittstelle und der Tests.

### 2. Kritisch — Unbefugtes Löschen fremder Pastes
**Befund:**  
`DELETE /pastes/{id}` verlangt keinerlei Berechtigung. Jeder, der eine ID kennt oder über `GET /pastes` erhält, kann den Eintrag irreversibel löschen. Das ist ein Verstoß gegen Integrität und Vertraulichkeit sowie gegen Art. 5 Abs. 1 lit. f DSGVO.

**Abhilfe:**
- Besitz-Token wie unter Punkt 1 einführen.
- `DELETE` nur mit gültigem Token oder nach Authentifizierung des Eigentümers zulassen (`handler/delete.go`, Tests in `handler/delete_test.go` anpassen).

### 3. Hoch — Fehlende Transportverschlüsselung (TLS)
**Befund:**  
`main.go` verwendet `http.ListenAndServe(":"+port, newApp())` und überträgt alle Daten im Klartext. Personenbezogene Inhalte können auf dem Transportweg mitgelesen werden (Art. 32 DSGVO).

**Abhilfe:**
- TLS aktivieren: `http.ListenAndServeTLS(":"+port, certFile, keyFile, newApp())` oder verbindlich einen TLS-terminierenden Reverse Proxy vorschreiben.
- Zusätzlich HSTS-Header setzen, sobald HTTPS aktiv ist (`main.go`, `corsMiddleware`).

### 4. Hoch — Kein automatischer Cleanup abgelaufener Pastes ohne Zugriff
**Befund:**  
Abgelaufene Pastes werden nur bei `Get` oder `List` physisch entfernt (`store/store.go`). Wird ein abgelaufener Paste nie wieder angefordert, bleibt er dauerhaft im Speicher. Das verletzt die Speicherbegrenzung nach Art. 5 Abs. 1 lit. e DSGVO.

**Abhilfe:**
- Hintergrund-Goroutine mit `time.Ticker` einführen, die regelmäßig abgelaufene Einträge aus der Map löscht, z. B. in `store.New()` oder `main.go` (`store/store.go`, `main.go`).
- Alternativ einen Scheduler im Store starten, der bei jeder Minute alle abgelaufenen IDs löscht.

### 5. Mittel — CORS `Access-Control-Allow-Origin: *`
**Befund:**  
`main.go` setzt pauschal `Access-Control-Allow-Origin: *`. In Kombination mit personenbezogenen Inhalten und fehlender Authentifizierung erlaubt dies beliebigen Websites einen unkontrollierten Zugriff aus dem Browser.

**Abhilfe:**
- Statt `*` eine explizite Allowlist vertrauenswürdiger Origins setzen (`main.go`, `corsMiddleware`).
- Sofern die API absichtlich öffentlich bleiben soll, zumindest dokumentieren und die Reichweite begrenzen.

### 6. Mittel — Fehlende Transparenz- und Rechtsgrundlagen-Dokumentation
**Befund:**  
Auch ohne UI muss der Betreiber die Datenverarbeitung nach Art. 13 DSGVO dokumentieren, insbesondere Zweck, Rechtsgrundlage, Speicherdauer und Betroffenenrechte. Im Repository ist keine solche Dokumentation sichtbar.

**Abhilfe:**
- `README.md` oder besser `PRIVACY.md` ergänzen mit Rechtsgrundlage (z. B. Art. 6 Abs. 1 lit. b DSGVO für die Vertragserfüllung und ggf. Art. 6 Abs. 1 lit. a für öffentliche Veröffentlichung), Speicherdauer, Löschkonzept, Rechte der betroffenen Personen.

---

## EU Cyber Resilience Act (CRA)

### 1. Hoch — Fehlende Authentifizierung/Autorisierung
**Befund:**  
Wie unter DSGVO dargestellt, fehlt jede Zugriffskontrolle. Das verletzt die CRA-Anforderungen an „security by design/default“ (unbefugter Zugriff, Manipulation und Löschung).

**Abhilfe:**  
Authentifizierung/API-Keys und Besitz-Token implementieren (siehe DSGVO Punkte 1 und 2).

### 2. Hoch — Keine Update-/Patch-Dokumentation, kein SBOM, keine dokumentierten Sicherheitseigenschaften
**Befund:**  
Es gibt keine sichtbare `SECURITY.md`, keine dokumentierte Update-/Patch-Prozedur und kein Software Bill of Materials (SBOM). `go.mod` enthält offenbar keine externen Abhängigkeiten (gut), aber die CRA verlangt dokumentierte Sicherheitsmerkmale und ein Inventar der Komponenten.

**Abhilfe:**
- `SECURITY.md` oder `README.md` um Sicherheitsmodell, Meldewege und Update-Prozess ergänzen.
- SBOM erzeugen, z. B. mit `go version -m` oder einem CycloneDX-Tool, und im Repo ablegen.
- Versions- und Changelog-Datei einführen.

### 3. Mittel — Fehlende Transportverschlüsselung
**Befund:**  
Wie unter DSGVO Punkt 3. CRA verlangt sichere Standardkonfigurationen; Klartext-HTTP ist ein Mangel.

**Abhilfe:**  
TLS aktivieren oder verbindlich extern terminieren lassen (`main.go`).

### 4. Niedrig — Fehlende Ressourcen- und Rate-Limits
**Befund:**  
Es gibt keine Begrenzung der Anzahl anlegbarer Pastes und keine Rate-Limits. Das kann zu Ressourcenüberlastung und Denial-of-Service führen.

**Abhilfe:**  
Rate-Limiting pro Client/IP und ggf. eine maximale Anzahl aktiver Pastes implementieren (`main.go`, `handler/create.go`).

---

## EU AI Act
Nicht anwendbar. Im Code ist keine KI-Funktion enthalten.

## Mandatory texts & UI
Für dieses reine Backend entfallen Cookie-Banner, Impressum und Accessibility-Pflichten. Die DSGVO-Transparenzpflicht (Art. 13) bleibt jedoch bestehen und ist unter DSGVO Punkt 6 adressiert.

## Accessibility
Entfällt mangels öffentlicher Web-UI.

---

**Zusammenfassung:**  
Der Dienst verarbeitet personenbezogene Daten ohne erkennbare Rechtsgrundlage für die öffentliche Zugänglichkeit und ohne jede Zugriffskontrolle. Jeder kann Pastes lesen und fremde Pastes löschen. Hinzu kommen fehlende Transportverschlüsselung und fehlender automatischer Ablauf-Cleanup. Dies sind fundamentale Datenschutz- und CRA-Verstöße, daher **BLOCKED**.