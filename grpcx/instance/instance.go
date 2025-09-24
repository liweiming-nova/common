package instance

import (
	"encoding/json"
)

// ServiceInstance 统一的服务实例信息
type ServiceInstance struct {
	Address      string            `json:"address"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	RegisteredAt int64             `json:"registered_at,omitempty"`
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
