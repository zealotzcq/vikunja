package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var (
	llmLogger *LLMLogger
)

type LLMLogger struct {
	enabled bool
	logDir  string
}

func GetLLMLogger() *LLMLogger {
	if llmLogger == nil {
		config := GetConfig()
		llmLogger = &LLMLogger{
			enabled: config.LLMLog,
			logDir:  "llmlog",
		}
	}
	return llmLogger
}

func (l *LLMLogger) LogRequest(provider, prompt string) error {
	if !l.enabled {
		return nil
	}

	timestamp := time.Now().Format("20060102_150405_000")
	filename := fmt.Sprintf("%s_req_%s.json", provider, timestamp)

	return l.writeJSONLog(filename, map[string]interface{}{
		"request": parseJSON(prompt),
	})
}

func (l *LLMLogger) LogResponse(provider, response string) error {
	if !l.enabled {
		return nil
	}

	timestamp := time.Now().Format("20060102_150405_000")
	filename := fmt.Sprintf("%s_resp_%s.json", provider, timestamp)

	return l.writeJSONLog(filename, map[string]interface{}{
		"response": parseJSON(response),
	})
}

func (l *LLMLogger) LogExchange(provider, request, response string) error {
	if !l.enabled {
		return nil
	}

	timestamp := time.Now().Format("20060102_150405_000")
	filename := fmt.Sprintf("%s_%s.json", provider, timestamp)

	return l.writeJSONLog(filename, map[string]interface{}{
		"request":  parseJSON(request),
		"response": parseJSON(response),
	})
}

func parseJSON(jsonStr string) interface{} {
	var result interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return jsonStr
	}
	return result
}

func (l *LLMLogger) writeJSONLog(filename string, data map[string]interface{}) error {
	if !l.enabled {
		return nil
	}

	if err := os.MkdirAll(l.logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal log: %w", err)
	}

	filePath := filepath.Join(l.logDir, filename)
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return fmt.Errorf("failed to write log file: %w", err)
	}

	return nil
}
