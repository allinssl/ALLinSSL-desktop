package kuocai

import "fmt"

type Response struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message"`
	Result  map[string]interface{} `json:"result"`
}

var pluginMeta = map[string]interface{}{
	"name": "kuocai", "description": "部署到括彩云", "version": "1.0.0", "author": "coclyun",
	"config":  map[string]interface{}{"username": "括彩云账号", "password": "括彩云密码"},
	"actions": []map[string]interface{}{{"name": "upload", "description": "部署到括彩云", "params": map[string]any{}}},
}

func Execute(action string, params map[string]interface{}) (*Response, error) {
	switch action {
	case "get_metadata":
		return &Response{Status: "success", Message: "插件信息", Result: pluginMeta}, nil
	case "list_actions":
		return &Response{Status: "success", Message: "支持的动作", Result: map[string]interface{}{"actions": pluginMeta["actions"]}}, nil
	case "upload":
		return Upload(params)
	default:
		return nil, fmt.Errorf("未知插件操作: %s", action)
	}
}
