package alicloud

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed metadata.json
var metadataJSON []byte

var pluginMeta map[string]interface{}

func init() {
	if err := json.Unmarshal(metadataJSON, &pluginMeta); err != nil {
		panic(fmt.Sprintf("解析元数据失败: %v", err))
	}
}

type Response struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message"`
	Result  map[string]interface{} `json:"result"`
}

// Execute 保留原部署逻辑，直接在软件进程中调用。
func Execute(action string, params map[string]interface{}) (*Response, error) {
	switch action {
	case "get_metadata":
		return &Response{Status: "success", Message: "插件信息", Result: pluginMeta}, nil
	case "list_actions":
		return &Response{Status: "success", Message: "支持的动作", Result: map[string]interface{}{"actions": pluginMeta["actions"]}}, nil
	case "cdn":
		return Cdn(params)
	case "dcdn":
		return Dcdn(params)
	case "oss":
		return Oss(params)
	case "waf":
		return Waf(params)
	case "esa":
		return Esa(params)
	case "cas":
		return Cas(params)
	default:
		return nil, fmt.Errorf("未知插件操作: %s", action)
	}
}
