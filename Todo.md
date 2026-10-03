# Todo

Offene Punkte, bewusst zurückgestellt. Der Jailer-Fix selbst ist erledigt und
verifiziert (Commit `8df51df`, VM `aivm` bootet im Jail, Gast-Schreibvorgänge
landen im Original).

## 1. Jailer-Config überlebt keinen Neustart

**Status:** offen, am 2026-10-03 zurückgestellt

### Problem

`vm.JailerConfig` lebt ausschließlich im Speicher: `NewManager` setzt Defaults,
`PUT /api/system/jailer` überschreibt sie zur Laufzeit (`SetJailerConfig`),
und `setup.Config` (die `settings.json`) hat kein Jailer-Feld. Nach **jedem**
Neustart des Dienstes — auch nach `firecrackmanager -setup` — ist `enabled`
wieder `false`.

Beobachtet am 2026-10-03: direkt nach dem zweiten `-setup`-Lauf meldete
`GET /api/system/jailer` `enabled: false`, obwohl der Jailer vorher aktiv
war.

### Folge

VMs starten ohne chroot/cgroups-Isolation. Sichtbar ist das nur am VM-Log-Eintrag
`Starting VM in standard mode (no jailer)`. Wer nach einem Neustart nicht in
die Jailer-Settings schaut, hält VMs mit Isolation für geschützt.

### Vorschlag

1. **In `settings.json` persistieren (empfohlen)** — ein `Jailer`-Block im
   `setup.Config` (`enabled`, `chroot_base`, `uid`, `gid`, `cgroup_version`,
   `netns`, `resource_limits`), den der Daemon beim Start lädt. Das
   API-Schema (`vm.JailerConfig`) bleibt unangetastet, die UI braucht keine
   Änderung. Überlebt Neustarts, Deployments und Reboots.
2. **Nur Pfad/UID beim Start laden, Toggle zur Laufzeit fragen** — kleiner
   Eingriff, aber die Isolation muss nach jedem Start per UI aktiviert werden;
   nur als Übergang sinnvoll.

Zusätzlich zu klären: Wer setzt die Datei, wenn der jailer im Deploy-Skript per
`-setup` provisioniert wird? Vermutlich `deploy.sh` analog zu
`enable_host_network_management`.

## 2. Jailer-Ausfall fällt still auf "ungejailed" zurück

**Status:** offen

`useJailer := m.jailerConfig != nil && m.jailerConfig.Enabled && m.IsJailerAvailable()`
(`internal/vm/vm.go`). Fehlt die jailer-Binary oder lässt sie sich nicht
ausführen, startet der VM **ohne** Jail, ohne Fehler — nur mit dem
Log-Eintrag `Starting VM in standard mode (no jailer)`.

Das ist ein stiller Fallback, den die Projekt-Regel (No-Fallback) verbietet.
Sinnvolle Optionen:

- Jailer-Pfad beim VM-Start einmal prüfen und bei `enabled: true` +
  nicht verfügbar den Start **abbrechen** (fail-fast).
- Oder mindestens im VM-Log als `error` statt `info` und in der VM-Übersicht
  sichtbar machen.

Beides zusammen: harter Fehler, damit die Isolation nicht unbemerkt
verschwindet.

## 3. Kleinere Beobachtungen aus derselben Analyse

- `/entropy` schlägt bei Firecracker 1.17 immer mit 400 fehl
  (`unknown field 'seed', expected 'rate_limiter'`). Aktuell nur eine
  Warnung pro Start; die Feldnamen bzw. der Endpunkt gehören angepasst.
- Die stdout-Pipe des VM-Prozesses wird nur bei geöffneter Web-Konsole
  gelesen (`GetConsoleStreams`). Ohne Client läuft das Boot-Log in den
  64-KB-Pipe-Puffer und firecracker blockiert dann. Betrifft jailed und
  unjailed gleich.
- Die Rootfs eines VM gehört nach dem ersten Jail-Start `uid 1000` (firecracker
  läuft als uid 1000 auf derselben inode). Kein Problem, aber unerwartet.
