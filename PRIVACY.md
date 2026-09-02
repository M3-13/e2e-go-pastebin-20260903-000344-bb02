# Datenschutzerklärung (PRIVACY)

Diese Datenschutzerklärung beschreibt, wie die Pastebin-REST-API (nachfolgend
„der Dienst“) personenbezogene Daten verarbeitet. Der Dienst ist eine
In-Memory-Pastebin-API: Inhalte werden ausschließlich im Arbeitsspeicher des
Prozesses gehalten und niemals persistent gespeichert.

## Verantwortliche Stelle

Verantwortlich im Sinne der Datenschutz-Grundverordnung (DSGVO) ist der
jeweilige Betreiber der konkreten Deployment-Instanz des Dienstes. Der Dienst
wird als Quellcode ausgeliefert und vom Betreiber selbst betrieben; der
Betreiber ist für den Betrieb und die Datenschutzkonformität seiner Instanz
verantwortlich.

**Kontakt:** Der Betreiber hinterlegt die Kontaktdaten (Name, Anschrift,
E-Mail) der für den Datenschutz zuständigen Stelle an der Instanz selbst.
Da der Quellcode ohne Betreiberkontakt verteilt wird, ist diese Stelle
instanzspezifisch zu ergänzen.

## Zweck der Verarbeitung

Der Dienst dient ausschließlich dazu, vom Nutzer eingereichte Textinhalte
(„Pastes“) für einen begrenzten Zeitraum oder bis zur manuellen Löschung über
einen Link bzw. eine zufällige ID bereitzustellen. Verarbeitet werden:

- der übermittelte Inhalt (`content`) sowie die optionale Sprachangabe
  (`language`),
- die gewählte Ablaufdauer (`expires_in_seconds`),
- die zufällig erzeugte Paste-ID (128 Bit Entropie),
- ein zufällig erzeugtes Lösch-Token (`delete_token`), das ausschließlich an
  den Ersteller zurückgegeben wird und die Löschung autorisiert,
- technische Zugriffsdaten (IP-Adresse, Zeitpunkt, aufgerufener Pfad), soweit
  der vorgeschaltete Server bzw. Reverse-Proxy diese protokolliert.

## Rechtsgrundlage

- **Art. 6 Abs. 1 lit. a DSGVO (Einwilligung)** für die linkbasierte
  Veröffentlichung: Der Nutzer entscheidet selbst, Inhalte zu erstellen, und
  erhält zu diesem Zweck einen Link bzw. eine ID, über die der Inhalt abgerufen
  werden kann. Durch das aktive Anlegen eines Pastes und das Teilen des Links
  willigt er in diese Bereitstellung ein.
- **Art. 6 Abs. 1 lit. b DSGVO (Vertragserfüllung)** für die technische
  Speicherung: Die Verarbeitung im Arbeitsspeicher ist erforderlich, um den
  angeforderten Dienst (Anlegen, Abrufen, Auflisten, Löschen) technisch
  zu erbringen.
- **Art. 6 Abs. 1 lit. f DSGVO (berechtigtes Interesse)** für die technische
  Speicherung und den Betrieb: das berechtigte Interesse liegt in der
  Gewährleistung der Funktionsfähigkeit, der IT-Sicherheit und der
  Missbrauchsabwehr des Dienstes.

## Speicherdauer

Alle Pastes werden **ausschließlich im Arbeitsspeicher (RAM)** gehalten. Es
erfolgt keine persistente Speicherung auf Datenträgern. Ein Paste wird entfernt:

- bei Ablauf der gewählten Frist (`expires_in_seconds`), oder
- sobald er über `DELETE /pastes/{id}` mit gültigem Lösch-Token gelöscht wird,
  oder
- beim Neustart bzw. Beenden des Prozesses (der gesamte Speicherinhalt geht
  verloren, da er nur im RAM existiert).

Ein Paste ohne Ablaufzeit bleibt bis zur manuellen Löschung oder bis zum
Prozessende im Arbeitsspeicher verfügbar.

## Löschkonzept

1. **Manuelle Löschung über Lösch-Token:** Beim Anlegen erhält der Ersteller
   ein einmaliges, kryptographisch zufälliges Lösch-Token (`delete_token`, 128
   Bit Entropie). Die Löschung erfolgt über `DELETE /pastes/{id}` mit dem
   Token im Header `X-Delete-Token`. Ohne gültiges Token wird die Löschung
   verweigert (401).
2. **Automatischer Ablauf:** Pastes mit `expires_in_seconds` werden nach Ablauf
   der Frist entfernt. Die Entfernung erfolgt sowohl beim Zugriff (lazy) als
   auch durch einen periodischen Hintergrund-Cleanup, der abgelaufene Einträge
   aus dem Speicher löscht.
3. **Vollständige Entfernung:** Gelöschte oder abgelaufene Einträge werden
   unmittelbar aus dem Store entfernt (nicht nur als ungültig markiert) und
   sind danach weder über `GET /pastes/{id}` noch über `GET /pastes` abrufbar.

## Betroffenenrechte

Betroffene Personen haben nach der DSGVO insbesondere das Recht auf:

- **Auskunft** (Art. 15 DSGVO) über die verarbeiteten Daten,
- **Berichtigung** (Art. 16 DSGVO) unrichtiger Daten,
- **Löschung** (Art. 17 DSGVO) — im Dienst über das Lösch-Token bzw. den
  Ablauf automatisch umgesetzt,
- **Einschränkung der Verarbeitung** (Art. 18 DSGVO),
- **Datenübertragbarkeit** (Art. 20 DSGVO),
- **Widerspruch** (Art. 21 DSGVO) und
- **Beschwerde** bei einer Aufsichtsbehörde (Art. 77 DSGVO).

Da Pastes keine dauerhaft gespeicherten personenbezogenen Daten enthalten und
ausschließlich im flüchtigen Arbeitsspeicher existieren, ist die wirksame
Ausübung dieser Rechte (insbesondere Löschung) jederzeit durch die oben
beschriebenen Löschmechanismen möglich. Zur Ausübung der Rechte wenden Sie sich
an die oben genannte Kontaktstelle des Betreibers.

## Empfänger / Übermittlung

Es findet keine automatisierte Übermittlung der Inhalte an Dritte statt.
Einzige Empfänger sind die technischen Systeme, über die der Dienst ausgeliefert
wird (vorgeschalteter Webserver / Reverse-Proxy / Load Balancer), die die
Anfragen lediglich transportieren.

## Sicherheit

Technische und organisatorische Maßnahmen zum Schutz der Daten sind in
`SECURITY.md` beschrieben. Hinweise zur Transportverschlüsselung (TLS) und zum
Zugriffsschutz finden sich dort sowie in der `README.md`.
