# Lokaler LLM-Server für batch-cost

## Übersicht
Der lokale LLM-Server ermöglicht die Nutzung eines kleinen Sprachmodells (noeum-1-nano-base) direkt auf dem lokalen Rechner, ohne externe Dienste wie LM Studio oder Ollama.

## Konfiguration

### Standardkonfiguration (pricing.toml)
```toml
[llm]
enabled = false
endpoint = "http://localhost:1234/v1"
model = ""
api_key = "lm-studio"
timeout_s = 30

[llm.local]
enabled = false
model_repo = "noeum/noeum-1-nano-base"
port = 2510
config_dir = ""
```

### Lokalen Modus aktivieren
```toml
[llm]
enabled = true

[llm.local]
enabled = true
```

## Verzeichnisstruktur
```
~/.config/batch-cost/
├── models/
│   └── noeum-noeum-1-nano-base/     # HuggingFace Weights (automatisch geladen)
├── venv/                            # Python virtual environment
├── server.py                        # FastAPI-Server
├── requirements.txt                 # Python-Abhängigkeiten
└── batch-cost-llm.pid              # PID des laufenden Servers
```

## Funktionsweise

### Beim ersten Start
1. Python virtual environment wird erstellt (mit `uv`)
2. Abhängigkeiten werden installiert (FastAPI, Transformers, Torch, etc.)
3. Modell wird von HuggingFace heruntergeladen (~paar GB)
4. FastAPI-Server startet auf Port 2510

### Bei jedem Start
1. Server wird auf Port 2510 gestartet
2. Modell wird geladen (schneller nach erstem Download)
3. `/health` Endpoint wird verfügbar
4. `/v1/chat/completions` Endpoint (OpenAI-kompatibel)

### Beim Beenden
1. Server wird automatisch beendet
2. PID-Datei wird gelöscht

## Technische Details

### Python-Server
- FastAPI-basierter Server
- OpenAI-kompatible API
- Automatische Modell-Download von HuggingFace
- Unterstützt CPU und GPU (CUDA/MPS)

### Modell
- **noeum-1-nano-base**: Kleines, effizientes Sprachmodell
- ~1.5B Parameter
- Geeignet für einfache Analyse-Aufgaben
- Läuft auf CPU (langsamer) oder GPU (schneller)

### Abhängigkeiten
- `uv`: Python-Paketmanager und virtual environment tool
- `torch`: PyTorch für ML-Inferenz
- `transformers`: HuggingFace Transformers für Modell-Loading
- `fastapi` + `uvicorn`: Web-Server

## Fehlerbehebung

### Server startet nicht
1. Prüfe ob `uv` installiert ist: `which uv`
2. Prüfe Logs: `~/.config/batch-cost/server.log`
3. Stelle sicher, dass Port 2510 frei ist

### Modell-Download fehlgeschlagen
1. Prüfe Internetverbindung
2. Prüfe Disk-Speicherplatz
3. Manueller Download: `huggingface-cli download noeum/noeum-1-nano-base`

### Langsame Inferenz
- Auf Mac: MPS wird automatisch erkannt
- Auf Linux/Windows: CUDA wird automatisch erkannt
- Ohne GPU: Läuft auf CPU (langsamer)

## Beispiel-Nutzung

```bash
# Mit lokaler LLM-Konfiguration
batch-cost --config test-local-llm.toml --job-id <job-id> --analyze

# Oder pricing.toml anpassen:
# [llm.local].enabled = true setzen
batch-cost --job-id <job-id> --analyze
```
