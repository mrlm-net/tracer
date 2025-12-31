package console

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// writeHTMLReport injects JSON events into the report template and writes file.
func writeHTMLReport(outPath string, events interface{}, stdout, stderr *os.File) error {
	jb, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}
	tplBytes, err := os.ReadFile("./public/report.html")
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}
	tplStr := string(tplBytes)
	if strings.Contains(tplStr, "<!--DATA-->") {
		tplStr = strings.Replace(tplStr, "<!--DATA-->", string(jb), 1)
	} else {
		script := fmt.Sprintf("<script id=\"__DATA__\" type=\"application/json\">%s</script>", jb)
		if strings.Contains(tplStr, "</body>") {
			tplStr = strings.Replace(tplStr, "</body>", script+"</body>", 1)
		} else {
			tplStr = tplStr + script
		}
	}
	if err := os.WriteFile(outPath, []byte(tplStr), 0644); err != nil {
		return fmt.Errorf("failed to write html: %w", err)
	}
	fmt.Fprintln(stdout, "Wrote HTML report to "+outPath)
	return nil
}
