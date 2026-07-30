package tunrepair

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var enableTunSetting = regexp.MustCompile(`(?i)("EnableTun"\s*:\s*)(true|false)`)

func EnableTunConfig(input []byte) ([]byte, bool, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(input, &root); err != nil {
		return nil, false, fmt.Errorf("解析 guiNConfig.json：%w", err)
	}
	tunRaw, exists := rawValueFold(root, "TunModeItem")
	if !exists {
		return nil, false, errors.New("guiNConfig.json 缺少 TunModeItem")
	}
	var tun map[string]json.RawMessage
	if err := json.Unmarshal(tunRaw, &tun); err != nil {
		return nil, false, fmt.Errorf("解析 TunModeItem：%w", err)
	}
	enabledRaw, exists := rawValueFold(tun, "EnableTun")
	if !exists {
		return nil, false, errors.New("guiNConfig.json 缺少 TunModeItem.EnableTun")
	}
	var enabled bool
	if err := json.Unmarshal(enabledRaw, &enabled); err != nil {
		return nil, false, errors.New("TunModeItem.EnableTun 不是布尔值")
	}
	if enabled {
		return append([]byte(nil), input...), false, nil
	}
	matches := enableTunSetting.FindAllSubmatchIndex(input, -1)
	if len(matches) != 1 {
		return nil, false, fmt.Errorf("EnableTun 字段数量异常：%d", len(matches))
	}
	match := matches[0]
	updated := make([]byte, 0, len(input))
	updated = append(updated, input[:match[4]]...)
	updated = append(updated, "true"...)
	updated = append(updated, input[match[5]:]...)
	return updated, true, nil
}

func CandidateConfigPaths(executablePath string, localAppData string) []string {
	var paths []string
	if executablePath != "" {
		paths = append(
			paths,
			filepath.Join(filepath.Dir(executablePath), "guiConfigs", "guiNConfig.json"),
		)
	}
	if localAppData != "" {
		paths = append(
			paths,
			filepath.Join(localAppData, "v2rayN", "guiConfigs", "guiNConfig.json"),
		)
	}
	return paths
}

func ConfigLocalPort(input []byte) (int, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(input, &root); err != nil {
		return 0, fmt.Errorf("解析 guiNConfig.json：%w", err)
	}
	inboundRaw, exists := rawValueFold(root, "Inbound")
	if !exists {
		return 10808, nil
	}
	var inbound []map[string]json.RawMessage
	if err := json.Unmarshal(inboundRaw, &inbound); err != nil || len(inbound) == 0 {
		return 0, errors.New("guiNConfig.json 的 Inbound 配置无效")
	}
	portRaw, exists := rawValueFold(inbound[0], "LocalPort")
	if !exists {
		return 10808, nil
	}
	var port int
	if err := json.Unmarshal(portRaw, &port); err != nil || port < 1 || port > 65529 {
		return 0, errors.New("guiNConfig.json 的 Inbound.LocalPort 无效")
	}
	return port, nil
}

func rawValueFold(values map[string]json.RawMessage, name string) (json.RawMessage, bool) {
	for key, value := range values {
		if strings.EqualFold(key, name) {
			return value, true
		}
	}
	return nil, false
}
