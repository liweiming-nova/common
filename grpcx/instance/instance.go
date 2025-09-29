package instance

import (
	"encoding/json"
	"strconv"
)

// ServiceInstance 统一的服务实例信息
type ServiceInstance struct {
	Address      string            `json:"address"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	RegisteredAt int64             `json:"registered_at,omitempty"`
	Score        float64           `json:"score,omitempty"`
}

// Encode 将实例编码为 JSON 字节
func Encode(si *ServiceInstance) ([]byte, error) {
	return json.Marshal(si)
}

// Decode 从 JSON 字节解析实例
func Decode(data []byte) (*ServiceInstance, error) {
	var si ServiceInstance
	if err := json.Unmarshal(data, &si); err != nil {
		return nil, err
	}
	return &si, nil
}

// ExtractAddress 优先按 JSON 解析，失败则将原值作为地址返回
func ExtractAddress(value string) string {
	if si, err := Decode([]byte(value)); err == nil && si != nil && si.Address != "" {
		return si.Address
	}
	return value
}

// ExtractScore 从 JSON 中解析 score，缺省返回 1
func ExtractScore(value string) float64 {
	if si, err := Decode([]byte(value)); err == nil && si != nil {
		if si.Score > 0 {
			return si.Score
		}
		// 兼容 Metadata 中的字符串 score
		if si.Metadata != nil {
			if v, ok := si.Metadata["score"]; ok && v != "" {
				// 尝试解析为 float64
				if f, err2 := strconv.ParseFloat(v, 64); err2 == nil && f > 0 {
					return f
				}
			}
		}
	}
	return 1
}
