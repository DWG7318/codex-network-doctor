package tunrepair

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

const (
	runtimePortSpan  = 7
	selectedPortSpan = 32
)

var localPortSetting = regexp.MustCompile(`(?i)("LocalPort"\s*:\s*)\d+`)

type PortRange struct {
	Start int
	End   int
}

func ChooseBasePort(
	current int,
	excluded []PortRange,
	listeners []Listener,
	scanStart int,
	scanEnd int,
) (int, bool, error) {
	if portBlockClear(current, runtimePortSpan, excluded, nil) {
		return current, false, nil
	}
	if scanStart < 1 || scanEnd > 65535 || scanStart > scanEnd {
		return 0, false, errors.New("端口扫描范围无效")
	}
	for candidate := scanStart; candidate <= scanEnd; candidate++ {
		if candidate+selectedPortSpan-1 > 65535 {
			break
		}
		if portBlockClear(candidate, selectedPortSpan, excluded, listeners) {
			return candidate, candidate != current, nil
		}
	}
	return 0, false, fmt.Errorf(
		"未在 %d-%d 找到连续 %d 个安全端口",
		scanStart,
		scanEnd,
		selectedPortSpan,
	)
}

func portBlockClear(
	base int,
	width int,
	excluded []PortRange,
	listeners []Listener,
) bool {
	if base < 1 || width < 1 || base+width-1 > 65535 {
		return false
	}
	end := base + width - 1
	for _, item := range excluded {
		if item.Start <= end && item.End >= base {
			return false
		}
	}
	for _, listener := range listeners {
		if listener.Port >= base && listener.Port <= end {
			return false
		}
	}
	return true
}

func SetConfigLocalPort(input []byte, port int) ([]byte, bool, error) {
	if port < 1 || port > 65529 {
		return nil, false, errors.New("新的 Inbound.LocalPort 无效")
	}
	current, err := ConfigLocalPort(input)
	if err != nil {
		return nil, false, err
	}
	if current == port {
		return append([]byte(nil), input...), false, nil
	}

	var root map[string]json.RawMessage
	if err := json.Unmarshal(input, &root); err != nil {
		return nil, false, fmt.Errorf("解析 guiNConfig.json：%w", err)
	}
	inboundRaw, exists := rawValueFold(root, "Inbound")
	if !exists {
		return nil, false, errors.New("guiNConfig.json 缺少 Inbound")
	}
	match := localPortSetting.FindSubmatchIndex(inboundRaw)
	if match == nil {
		return nil, false, errors.New("guiNConfig.json 缺少 Inbound[0].LocalPort")
	}
	inboundOffset := bytes.Index(input, inboundRaw)
	if inboundOffset < 0 {
		return nil, false, errors.New("无法定位 guiNConfig.json 的 Inbound")
	}

	valueStart := inboundOffset + match[3]
	valueEnd := inboundOffset + match[1]
	updated := make([]byte, 0, len(input)+6)
	updated = append(updated, input[:valueStart]...)
	updated = strconv.AppendInt(updated, int64(port), 10)
	updated = append(updated, input[valueEnd:]...)
	return updated, true, nil
}
