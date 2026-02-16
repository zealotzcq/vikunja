package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	llmLogger *LLMLogger
)

type LLMLogger struct {
	enabled    bool
	logDir     string
	briefMode  bool
	counter    uint64
	counterMux sync.Mutex
}

func GetLLMLogger() *LLMLogger {
	if llmLogger == nil {
		config := GetConfig()
		llmLogger = &LLMLogger{
			enabled:   config.LLMLog,
			logDir:    config.LLMLogPath,
			briefMode: config.LLMLogBriefMode,
			counter:   0,
		}
	}
	return llmLogger
}

func (l *LLMLogger) nextCounter() uint64 {
	l.counterMux.Lock()
	defer l.counterMux.Unlock()
	l.counter++
	return l.counter
}

func (l *LLMLogger) LogRequest(provider, prompt string) error {
	if !l.enabled {
		return nil
	}

	timestamp := time.Now().Format("20060102_150405_000")
	counter := l.nextCounter()
	filename := fmt.Sprintf("%s_req_%s_%04d.json", provider, timestamp, counter)

	requestData := parseJSON(prompt)

	if l.briefMode {
		if reqMap, ok := requestData.(map[string]interface{}); ok {
			if requestInner, ok := reqMap["request"].(map[string]interface{}); ok {
				delete(requestInner, "tools")
			}
		}
	}

	return l.writeJSONLog(filename, map[string]interface{}{
		"request": requestData,
	})
}

func (l *LLMLogger) LogResponse(provider, response string) error {
	if !l.enabled {
		return nil
	}

	timestamp := time.Now().Format("20060102_150405_000")
	counter := l.nextCounter()
	filename := fmt.Sprintf("%s_resp_%s_%04d.json", provider, timestamp, counter)

	return l.writeJSONLog(filename, map[string]interface{}{
		"response": parseJSON(response),
	})
}

func (l *LLMLogger) LogExchange(provider, request, response string) error {
	if !l.enabled {
		return nil
	}

	timestamp := time.Now().Format("20060102_150405_000")
	counter := l.nextCounter()
	filename := fmt.Sprintf("%s_%s_%04d.json", provider, timestamp, counter)

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
