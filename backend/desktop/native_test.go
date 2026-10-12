package desktop

import (
	"ALLinSSL/backend/internal/workflow"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type nativeResult struct {
	Status  bool            `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
	Count   int             `json:"count"`
}

func nativeCall(t *testing.T, engine *Runtime, path string, fields url.Values, expected int) nativeResult {
	t.Helper()
	method := "GET"
	if fields != nil {
		method = "POST"
	}
	status, _, body := engine.Request(method, path, http.Header{"content-type": {"application/x-www-form-urlencoded"}, "cookie": {"损坏的旧会话"}}, []byte(fields.Encode()))
	if status != expected {
		t.Fatalf("%s 状态 %d，响应 %s", path, status, body)
	}
	var result nativeResult
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestNativeLifecycle(t *testing.T) {
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(t.TempDir(), "中文 空格目录")
	var engine Runtime
	defer func() { engine.Stop(); _ = os.Chdir(original) }()
	entry, err := engine.Start(directory)
	if err != nil || entry != "/login" {
		t.Fatalf("首次启动：%s %v", entry, err)
	}
	if _, err = engine.Start(directory); err == nil {
		t.Fatal("不得重复启动调度")
	}
	state := nativeCall(t, &engine, "/v1/login/state", nil, 200)
	if !bytes.Contains(state.Data, []byte(`"setup":true`)) {
		t.Fatal("新数据库没有进入可选密码设置")
	}
	if !nativeCall(t, &engine, "/v1/login/skip_setup", url.Values{}, 200).Status {
		t.Fatal("首次跳过失败")
	}
	if !nativeCall(t, &engine, "/v1/setting/save_setting", url.Values{"timeout": {"1"}, "password_enabled": {"false"}}, 200).Status {
		t.Fatal("设置保存失败")
	}
	time.Sleep(1100 * time.Millisecond)
	state = nativeCall(t, &engine, "/v1/login/state", nil, 200)
	if !bytes.Contains(state.Data, []byte(`"unlocked":true`)) {
		t.Fatal("不设密码时不应锁定")
	}
	engine.Stop()
	entry, err = engine.Start(directory)
	if err != nil || entry != "/" {
		t.Fatalf("跳过密码没有持久化：%s %v", entry, err)
	}
	password := md5.Sum([]byte("进程内验收密码_bt_all_in_ssl"))
	digest := hex.EncodeToString(password[:])
	if !nativeCall(t, &engine, "/v1/login/sign", url.Values{"setup": {"true"}, "password": {digest}}, 200).Status {
		t.Fatal("软件密码设置失败")
	}
	nativeCall(t, &engine, "/v1/setting/save_setting", url.Values{"timeout": {"3600"}, "password_enabled": {"true"}}, 200)
	if nativeCall(t, &engine, "/v1/login/skip_setup", url.Values{}, 500).Status {
		t.Fatal("不得绕过已有密码")
	}
	if nativeCall(t, &engine, "/v1/setting/save_setting", url.Values{"timeout": {"3600"}, "password_enabled": {"false"}}, 500).Status {
		t.Fatal("关闭保护必须验证密码")
	}
	root := nativeCall(t, &engine, "/v1/private_ca/create_root_ca", url.Values{"name": {"进程内根CA"}, "cn": {"Native Test CA"}, "o": {"ALLinSSL"}, "c": {"CN"}, "algorithm": {"rsa"}, "key_length": {"2048"}, "valid_days": {"7"}}, 200)
	if !root.Status {
		t.Fatal(root.Message)
	}
	roots := nativeCall(t, &engine, "/v1/private_ca/get_ca_list", url.Values{"p": {"1"}, "limit": {"10"}, "level": {"root"}}, 200)
	var authorities []struct {
		ID int `json:"id"`
	}
	if err = json.Unmarshal(roots.Data, &authorities); err != nil || len(authorities) != 1 {
		t.Fatalf("CA列表：%s %v", roots.Data, err)
	}
	id, _ := json.Marshal(authorities[0].ID)
	status, _, content := engine.Request("GET", "/v1/private_ca/download_cert?id="+string(id)+"&type=ca", http.Header{}, nil)
	if status != 200 || !bytes.HasPrefix(content, []byte("PK")) {
		t.Fatal("CA导出失败")
	}
	status, _, content = engine.Request("GET", "/v1/setting/download_data", http.Header{}, nil)
	if status != 200 || !bytes.HasPrefix(content, []byte("SQLite format 3")) {
		t.Fatal("数据备份失败")
	}
	var upload bytes.Buffer
	writer := multipart.NewWriter(&upload)
	file, err := writer.CreateFormFile("file", "中文监控模板.json")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = file.Write([]byte(`[{"name":"进程内导入","target":"127.0.0.1","type":"https","port":443,"active":"0"}]`))
	_ = writer.Close()
	status, _, content = engine.Request("POST", "/v1/monitor/file_add_monitor", http.Header{"Content-Type": {writer.FormDataContentType()}}, upload.Bytes())
	if status != 200 || !bytes.Contains(content, []byte(`"status":true`)) {
		t.Fatalf("文件上传：%d %s", status, content)
	}
	settings := nativeCall(t, &engine, "/v1/setting/get_setting", url.Values{}, 200)
	var options map[string]any
	_ = json.Unmarshal(settings.Data, &options)
	if len(options) != 3 || options["password_enabled"] != true {
		t.Fatalf("设置字段：%s", settings.Data)
	}
	nativeCall(t, &engine, "/v1/setting/save_setting", url.Values{"timeout": {"1"}}, 200)
	time.Sleep(1100 * time.Millisecond)
	nativeCall(t, &engine, "/v1/cert/get_list", url.Values{"p": {"1"}}, 401)
	if !nativeCall(t, &engine, "/v1/login/sign", url.Values{"password": {digest}}, 200).Status {
		t.Fatal("超时重新解锁失败")
	}
	if !nativeCall(t, &engine, "/v1/setting/save_setting", url.Values{"timeout": {"3600"}, "password_enabled": {"false"}, "current_password": {digest}}, 200).Status {
		t.Fatal("关闭保护失败")
	}
	// 模拟完整退出时仍有手动任务，退出必须等它完成。
	workflow.DesktopExecutions.Add(1)
	go func() { time.Sleep(180 * time.Millisecond); workflow.DesktopExecutions.Done() }()
	started := time.Now()
	engine.Stop()
	if time.Since(started) < 150*time.Millisecond {
		t.Fatal("退出未等待工作流")
	}
	status, _, _ = engine.Request("GET", "/v1/login/state", http.Header{}, nil)
	if status != 503 {
		t.Fatal("退出后仍可调用业务")
	}
	entry, err = engine.Start(directory)
	if err != nil || entry != "/" {
		t.Fatalf("关闭保护没有持久化：%s %v", entry, err)
	}
	if err = engine.ResetPassword(digest); err != nil {
		t.Fatal(err)
	}
	if !nativeCall(t, &engine, "/v1/login/sign", url.Values{"setup": {"true"}, "password": {digest}}, 200).Status {
		t.Fatal("重新启用保护失败")
	}
	engine.Stop()
	entry, err = engine.Start(directory)
	if err != nil || entry != "/login" {
		t.Fatalf("启用保护没有持久化：%s %v", entry, err)
	}
	if !nativeCall(t, &engine, "/v1/login/sign", url.Values{"password": {digest}}, 200).Status {
		t.Fatal("完整启动解锁失败")
	}
	if !nativeCall(t, &engine, "/v1/setting/restart", url.Values{}, 200).Status {
		t.Fatal("重新加载配置失败")
	}
	nativeCall(t, &engine, "/v1/cert/get_list", url.Values{"p": {"1"}}, 401)
	if !nativeCall(t, &engine, "/v1/login/sign", url.Values{"password": {digest}}, 200).Status {
		t.Fatal("配置重载后解锁失败")
	}
	roots = nativeCall(t, &engine, "/v1/private_ca/get_ca_list", url.Values{"p": {"1"}, "limit": {"10"}, "level": {"root"}}, 200)
	if roots.Count != 1 {
		t.Fatal("重启丢失CA数据")
	}
	stored, err := os.ReadFile(filepath.Join(directory, "data", "settings.db"))
	if err != nil || bytes.Contains(stored, []byte("进程内验收密码")) {
		t.Fatal("数据库存储失败或含明文密码")
	}
}
