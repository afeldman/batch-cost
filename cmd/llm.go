package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/afeldman/batch-cost/internal/llm"
	"github.com/afeldman/batch-cost/internal/pricing"
	"github.com/spf13/cobra"
)

var llmCmd = &cobra.Command{
	Use:   "llm",
	Short: "Lokales LLM verwalten",
}

var llmDownloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Lokales LLM-Modell herunterladen",
	RunE:  runLLMDownload,
}

var llmStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Status des lokalen LLM anzeigen",
	RunE:  runLLMStatus,
}

func init() {
	llmCmd.AddCommand(llmDownloadCmd)
	llmCmd.AddCommand(llmStatusCmd)
	rootCmd.AddCommand(llmCmd)
}

func runLLMDownload(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	_, _, llmCfg, err := pricing.LoadOptions(flagConfig)
	if err != nil {
		return err
	}

	if !llmCfg.Local.Enabled {
		fmt.Println("⚠  llm.local.enabled = false in pricing.toml")
		fmt.Println("   Setze [llm.local] enabled = true um das lokale Modell zu nutzen.")
	}

	if llmCfg.Local.ModelRepo == "" {
		return fmt.Errorf("kein Modell konfiguriert — bitte [llm.local] model_repo in pricing.toml setzen")
	}

	mgr := llm.NewManager(llmCfg.Local)
	return mgr.Download(ctx)
}

func runLLMStatus(cmd *cobra.Command, args []string) error {
	_, _, llmCfg, err := pricing.LoadOptions(flagConfig)
	if err != nil {
		return err
	}

	cfgDir := llmCfg.Local.ConfigDir
	if cfgDir == "" {
		cfgDir = filepath.Join(os.Getenv("HOME"), ".config", "batch-cost")
	}

	fmt.Println("=== LLM Status ===")
	fmt.Println()

	// Konfiguration
	fmt.Printf("%-20s %v\n", "Local enabled:", llmCfg.Local.Enabled)
	fmt.Printf("%-20s %s\n", "Modell:", llmCfg.Local.ModelRepo)
	fmt.Printf("%-20s %d\n", "Port:", llmCfg.Local.Port)
	fmt.Printf("%-20s %s\n", "Config-Dir:", cfgDir)
	fmt.Println()

	// Venv
	venvDir := filepath.Join(cfgDir, "venv")
	if _, err := os.Stat(venvDir); err == nil {
		fmt.Println("✓  Python-Umgebung vorhanden")
	} else {
		fmt.Println("✗  Python-Umgebung fehlt  (batch-cost llm download)")
	}

	// Modell
	if llmCfg.Local.ModelRepo != "" {
		modelDir := filepath.Join(cfgDir, "models",
			replaceSlash(llmCfg.Local.ModelRepo))
		if _, err := os.Stat(modelDir); err == nil {
			fmt.Printf("✓  Modell vorhanden: %s\n", modelDir)
		} else {
			fmt.Printf("✗  Modell fehlt       (batch-cost llm download)\n")
		}
	}

	// Server läuft?
	mgr := llm.NewManager(llmCfg.Local)
	if mgr.IsHealthy() {
		fmt.Printf("✓  Server läuft auf Port %d\n", llmCfg.Local.Port)
	} else {
		fmt.Printf("–  Server nicht aktiv  (wird beim Start von batch-cost --analyze gestartet)\n")
	}

	return nil
}

func replaceSlash(s string) string {
	result := ""
	for _, c := range s {
		if c == '/' {
			result += "-"
		} else {
			result += string(c)
		}
	}
	return result
}
