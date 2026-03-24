package llm

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Manager verwaltet den Python-LLM-Subprocess.
type Manager struct {
	cfg     LocalConfig
	proc    *exec.Cmd
	cfgDir  string
}

// NewManager erstellt einen neuen Subprocess-Manager.
func NewManager(cfg LocalConfig) *Manager {
	dir := cfg.ConfigDir
	if dir == "" {
		dir = filepath.Join(os.Getenv("HOME"), ".config", "batch-cost")
	}
	return &Manager{cfg: cfg, cfgDir: dir}
}



// IsHealthy prüft ob der Server auf Port 2510 antwortet.
func (m *Manager) IsHealthy() bool {
	url := fmt.Sprintf("http://localhost:%d/health", m.cfg.Port)
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 200
}

// setup installiert uv-Umgebung + Python-Dependencies falls nötig.
func (m *Manager) setup(ctx context.Context) error {
	if err := os.MkdirAll(filepath.Join(m.cfgDir, "models"), 0755); err != nil {
		return err
	}

	serverPy := filepath.Join(m.cfgDir, "server.py")
	if _, err := os.Stat(serverPy); os.IsNotExist(err) {
		if err := os.WriteFile(serverPy, []byte(serverPyContent(m.cfg.ModelRepo, m.cfg.Port)), 0644); err != nil {
			return err
		}
	}

	reqTxt := filepath.Join(m.cfgDir, "requirements.txt")
	if _, err := os.Stat(reqTxt); os.IsNotExist(err) {
		if err := os.WriteFile(reqTxt, []byte(requirementsTxt), 0644); err != nil {
			return err
		}
	}

	// uv venv erstellen falls nicht vorhanden
	venvDir := filepath.Join(m.cfgDir, "venv")
	if _, err := os.Stat(venvDir); os.IsNotExist(err) {
		fmt.Println("⚙  LLM: erstelle Python-Umgebung (einmalig)...")
		cmd := exec.CommandContext(ctx, "uv", "venv", venvDir)
		cmd.Dir = m.cfgDir
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("uv venv: %s: %w", out, err)
		}
	}

	// Dependencies installieren
	fmt.Println("⚙  LLM: installiere Dependencies (einmalig)...")
	cmd := exec.CommandContext(ctx, "uv", "pip", "install",
		"--python", filepath.Join(venvDir, "bin", "python"),
		"-r", reqTxt)
	cmd.Dir = m.cfgDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("uv pip install: %s: %w", out, err)
	}

	return nil
}

// Download installiert venv + Dependencies und lädt das Modell herunter.
// Stdout/Stderr sind sichtbar (für expliziten Download-Befehl).
func (m *Manager) Download(ctx context.Context) error {
	if err := m.setup(ctx); err != nil {
		return err
	}

	modelDir := filepath.Join(m.cfgDir, "models", strings.ReplaceAll(m.cfg.ModelRepo, "/", "-"))
	if _, err := os.Stat(modelDir); err == nil {
		fmt.Printf("✓  Modell bereits vorhanden: %s\n", modelDir)
		return nil
	}

	fmt.Printf("⬇  Lade Modell %s herunter nach %s...\n", m.cfg.ModelRepo, modelDir)
	fmt.Println("   (Das kann einige Minuten dauern — Modell wird einmalig gespeichert)")

	venvPython := filepath.Join(m.cfgDir, "venv", "bin", "python")
	script := fmt.Sprintf(`
from huggingface_hub import snapshot_download
import sys
path = snapshot_download(repo_id="%s", local_dir="%s")
print(f"✓  Modell gespeichert: {path}")
`, m.cfg.ModelRepo, modelDir)

	cmd := exec.CommandContext(ctx, venvPython, "-c", script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// StartDaemon startet den Server als Daemon (überlebt batch-cost).
// Wird beim nächsten Aufruf via IsHealthy wiederverwendet.
func (m *Manager) StartDaemon(ctx context.Context) error {
    if err := m.setup(ctx); err != nil {
        return err
    }
    return m.start(ctx, true)
}

// EnsureReady wird intern genutzt (z.B. von llm start-Befehl).
func (m *Manager) EnsureReady(ctx context.Context) error {
    if m.IsHealthy() {
        return nil
    }
    if err := m.setup(ctx); err != nil {
        return fmt.Errorf("llm setup: %w", err)
    }
    return m.start(ctx, true)
}

// start startet den Python-Server.
// daemon=true: Prozess überlebt batch-cost (kein Kill beim Exit).
func (m *Manager) start(ctx context.Context, daemon bool) error {
	venvPython := filepath.Join(m.cfgDir, "venv", "bin", "python")
	serverPy := filepath.Join(m.cfgDir, "server.py")

	cmd := exec.Command(venvPython, serverPy)
	cmd.Env = append(os.Environ(),
		"MODEL_REPO="+m.cfg.ModelRepo,
		"MODEL_DIR="+filepath.Join(m.cfgDir, "models"),
		"PORT="+strconv.Itoa(m.cfg.Port),
	)
	cmd.Dir = m.cfgDir
	logFile, _ := os.Create(filepath.Join(m.cfgDir, "server.log"))
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if daemon {
		// Prozess vom Parent-Prozess lösen → überlebt batch-cost
		cmd.SysProcAttr = daemonSysProcAttr()
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("python server start: %w", err)
	}

	_ = os.WriteFile(
		filepath.Join(m.cfgDir, "batch-cost-llm.pid"),
		[]byte(strconv.Itoa(cmd.Process.Pid)),
		0644,
	)

	if !daemon {
		m.proc = cmd // nur tracken wenn nicht Daemon
	}

	// Warten bis Server bereit (max 90s — Modell-Loading kann dauern)
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		if m.IsHealthy() {
			fmt.Println("✓  LLM-Server bereit (läuft im Hintergrund)")
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	// Timeout — Prozess killen
	_ = cmd.Process.Kill()
	_ = os.Remove(filepath.Join(m.cfgDir, "batch-cost-llm.pid"))
	return fmt.Errorf("LLM-Server Timeout — Log: %s", filepath.Join(m.cfgDir, "server.log"))
}

// serverPyContent generiert den Python-Server-Code.
func serverPyContent(modelRepo string, port int) string {
	return fmt.Sprintf(`#!/usr/bin/env python3
"""
batch-cost lokaler LLM-Server
OpenAI-kompatibler FastAPI-Server auf Port %d
Wird von batch-cost automatisch gestartet und beendet.
"""
import os
import torch
from fastapi import FastAPI
from pydantic import BaseModel
from transformers import AutoModelForCausalLM, AutoTokenizer
import uvicorn

MODEL_REPO = os.environ.get("MODEL_REPO", "%s")
MODEL_DIR  = os.environ.get("MODEL_DIR", os.path.expanduser("~/.config/batch-cost/models"))
PORT       = int(os.environ.get("PORT", "%d"))

model_path = os.path.join(MODEL_DIR, MODEL_REPO.replace("/", "-"))

app = FastAPI()
tokenizer = None
model = None

@app.on_event("startup")
async def load_model():
    global tokenizer, model
    print(f"Lade Modell: {MODEL_REPO}")

    # Download falls nicht vorhanden
    if not os.path.exists(model_path):
        print(f"Lade von HuggingFace herunter nach {model_path}...")
        from huggingface_hub import snapshot_download
        snapshot_download(repo_id=MODEL_REPO, local_dir=model_path)

    tokenizer = AutoTokenizer.from_pretrained(model_path, trust_remote_code=True)
    model = AutoModelForCausalLM.from_pretrained(
        model_path,
        trust_remote_code=True,
        torch_dtype=torch.float16 if torch.cuda.is_available() else torch.float32,
    )
    model.eval()
    print("Modell geladen.")

@app.get("/health")
async def health():
    return {"status": "ok", "model": MODEL_REPO}

class Message(BaseModel):
    role: str
    content: str

class ChatRequest(BaseModel):
    model: str = ""
    messages: list[Message]
    stream: bool = False

class ChatResponse(BaseModel):
    choices: list[dict]

@app.post("/v1/chat/completions")
async def chat(req: ChatRequest):
    prompt = req.messages[-1].content if req.messages else ""
    inputs = tokenizer(prompt, return_tensors="pt")
    with torch.no_grad():
        outputs = model.generate(
            **inputs,
            max_new_tokens=256,
            temperature=0.7,
            do_sample=True,
        )
    response = tokenizer.decode(outputs[0][inputs["input_ids"].shape[1]:], skip_special_tokens=True)
    return {
        "choices": [{"message": {"role": "assistant", "content": response}}]
    }

if __name__ == "__main__":
    uvicorn.run(app, host="127.0.0.1", port=PORT, log_level="warning")
`, port, modelRepo, port)
}

const requirementsTxt = `fastapi>=0.110.0
uvicorn>=0.29.0
transformers>=4.40.0
torch>=2.2.0
huggingface_hub>=0.22.0
pydantic>=2.0.0
tiktoken>=0.7.0
sentencepiece>=0.2.0
`
