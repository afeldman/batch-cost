#!/usr/bin/env bash
# Beispiel-Skript für batch-cost

echo "=== batch-cost Demo ==="
echo ""

# 1. Hilfe anzeigen
echo "1. Hilfe anzeigen:"
./batch-cost --help | head -20
echo ""

# 2. Test der Kostenberechnung
echo "2. Test der Kostenberechnung (lib/pricing.sh):"
source lib/pricing.sh
echo "calc_cost 3600 2 4096:"
calc_cost 3600 2 4096
echo ""

# 3. Test der Formatierung
echo "3. Test der Formatierung:"
echo "format_duration 3665: $(format_duration 3665)"
echo ""

# 4. Test der Ausgabe-Funktionen
echo "4. Test der Ausgabe-Funktionen:"
source lib/output.sh
echo "header 'Test Header':"
header "Test Header"
echo ""
echo "label 'Key' 'Value':"
label "Key" "Value"
echo ""
echo "cost_line 'CPU Cost' '0.1234':"
cost_line "CPU Cost" "0.1234"
echo ""
echo "warn 'Test Warning':"
warn "Test Warning"
echo ""

# 5. Struktur anzeigen
echo "5. Projektstruktur:"
ls -la
echo ""
echo "lib/:"
ls -la lib/
echo ""
echo "providers/:"
ls -la providers/
echo ""

echo "=== Demo abgeschlossen ==="
echo ""
echo "Verwendung:"
echo "  ./batch-cost --help                    # Hilfe anzeigen"
echo "  ./batch-cost --job-id <job-id>         # Kosten schätzen"
echo "  ./batch-cost --job-name <job-name>     # Echte Kosten"
echo "  ./batch-cost                           # Interaktiver Modus (mit gum)"
