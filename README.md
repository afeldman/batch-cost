# batch-cost Go CLI

Ein modulares CLI-Tool zur Kostenanalyse von Batch-Jobs in der Cloud, implementiert in Go.

## Features

- **Mode 1: Kosten-Schätzung** – Berechnet Kosten basierend auf Job-Metadaten (vCPU, Memory, Laufzeit)
- **Mode 2: Echte Kosten** – Abfrage von AWS Cost Explorer mit Job-Tags
- **Multi-Provider** – Erweiterbar für AWS, GCP, Azure (aktuell nur AWS)
- **Interaktiver Modus** – Mit `huh` für schöne Terminal-UI
- **JSON Output** – Maschinenlesbare Ausgabe für Automatisierung
- **Loading Spinner** – Bubble Tea Spinner für API-Calls
- **Farbige Ausgabe** – Lipgloss für schöne Terminal-Outputs

## Installation

### Option 1: Binary herunterladen (bald verfügbar)

```bash
# Kommt bald...
```

### Option 2: Von Source bauen

```bash
# 1. Repository klonen
git clone <repository-url>
cd batch-cost

# 2. Go 1.22+ installieren
# macOS
brew install go

# Ubuntu/Debian
sudo apt-get install -y golang

# 3. Dependencies holen und bauen
go mod tidy
go build -o batch-cost-go ./main.go

# 4. AWS CLI konfigurieren (falls noch nicht geschehen)
aws configure
```

## Verwendung

### Grundlegende Befehle

```bash
# Hilfe anzeigen
./batch-cost-go --help

# Kosten für einen Job schätzen
./batch-cost-go --job-id <job-id>

# Echte Kosten via Cost Explorer (benötigt Tag: BatchJob=<name>)
./batch-cost-go --job-name <job-name>

# JSON Output
./batch-cost-go --job-id <job-id> --json

# Andere AWS Region
./batch-cost-go --job-id <job-id> --region us-east-1

# Anderes AWS Profil
./batch-cost-go --job-id <job-id> --profile production
```

### Interaktiver Modus

Wenn keine Argumente angegeben werden, startet der interaktive Modus:

```bash
./batch-cost-go
```

## Konfiguration

### Umgebungsvariablen

```bash
# AWS Konfiguration
export AWS_REGION=eu-central-1
export AWS_PROFILE=production

# Preise anpassen (AWS Fargate on-demand, eu-central-1)
export PRICE_PER_VCPU_HOUR=0.04048
export PRICE_PER_GB_HOUR=0.004445
```

### TOML Konfiguration

Erstelle eine `pricing.toml` Datei:

```toml
[pricing]
price_vcpu_hour = 0.04048
price_gb_hour = 0.004445
```

Speicherorte (in dieser Reihenfolge):
1. `./pricing.toml` (aktuelles Verzeichnis)
2. `~/.config/batch-cost/pricing.toml`

### AWS Cost Explorer Tags

Für Mode 2 (echte Kosten) müssen Jobs mit einem Tag versehen werden:

```json
{
  "Tags": {
    "BatchJob": "<job-name>"
  }
}
```

## Architektur

```
batch-cost/
├── go.mod                 # Go Module Definition
├── go.sum                 # Dependency Lock
├── main.go               # Entry Point
├── cmd/
│   └── root.go           # Cobra CLI Definition
├── internal/
│   ├── providers/
│   │   ├── provider.go   # Provider Interface
│   │   └── aws/
│   │       ├── batch.go  # AWS Batch Integration
│   │       └── ce.go     # AWS Cost Explorer Integration
│   ├── pricing/
│   │   └── calc.go       # Kostenberechnung
│   └── ui/
│       ├── output.go     # Lipgloss Output
│       └── spinner.go    # Bubble Tea Spinner
└── README.md
```

### Neue Provider hinzufügen

1. Implementiere das `Provider` Interface aus `internal/providers/provider.go`
2. Erstelle einen neuen Provider im `internal/providers/` Verzeichnis
3. Füge den Provider in `cmd/root.go` hinzu

## Beispiele

### Beispiel 1: Kosten-Schätzung

```bash
$ ./batch-cost-go --job-id abc123-def456

=== Batch Job Cost Report ===

Job ID:           abc123-def456
Job Name:         data-processing-job
Status:           SUCCEEDED
Duration:         7200s (2.0000h)  →  2h 00m 00s
vCPU:             4
Memory:           8192 MB
──────────────────────────────────────────

--- Estimate (aws) ---

CPU Cost:         $0.3238
Memory Cost:      $0.1422
──────────────────────────────────────────
Total:            $0.4660
```

### Beispiel 2: JSON Output

```bash
$ ./batch-cost-go --job-id abc123-def456 --json
{
  "job": {
    "JobID": "abc123-def456",
    "JobName": "data-processing-job",
    "Status": "SUCCEEDED",
    "InProgress": false,
    "DurationSec": 7200,
    "VCPU": 4,
    "MemoryMB": 8192
  },
  "cost": {
    "DurationSec": 7200,
    "DurationH": 2,
    "CPUCost": 0.32384,
    "MemCost": 0.14224000000000002,
    "Total": 0.46608000000000004
  }
}
```

### Beispiel 3: Interaktiver Modus

```bash
$ ./batch-cost-go
? batch-cost — Was möchtest du tun? ›
  Kosten schätzen (Job ID)
  Echte Kosten (Cost Explorer)

? AWS Batch Job ID › xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

## Voraussetzungen

- **Go** 1.22+
- **AWS CLI** v2+ (für AWS Credentials)
- **AWS IAM Permissions**:
  - `batch:DescribeJobs`
  - `ce:GetCostAndUsage`

## Fehlerbehandlung

- Fehlende AWS Credentials werden erkannt
- AWS API-Fehler werden angezeigt
- Ungültige Job-IDs führen zu klaren Fehlermeldungen
- Type Assertion Fehler werden abgefangen

## Lizenz

MIT
