package plugin

import "testing"

func TestBuiltins(t *testing.T) {
	plugins, err := scanPlugins(t.TempDir())
	if err != nil || len(plugins) != 2 {
		t.Fatalf("内置插件：%v %v", plugins, err)
	}
	for _, entry := range plugins {
		metadata, err := runPlugin(entry.Path, Request{Action: "get_metadata"})
		if err != nil || metadata.Status != "success" || len(entry.Actions) == 0 {
			t.Fatalf("内置插件元数据：%s %v", entry.Name, err)
		}
		if _, err = runPlugin(entry.Path, Request{Action: "不存在的操作"}); err == nil {
			t.Fatal("插件不得执行未知操作")
		}
	}
}
