package desktop

import (
	"ALLinSSL/backend/internal/workflow"
	"ALLinSSL/backend/migrations"
	"ALLinSSL/backend/public"
	"ALLinSSL/backend/scheduler"
	"ALLinSSL/backend/server"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"
)

// Runtime 直接处理进程内业务调用，不创建监听端口或子进程。
type Runtime struct {
	mu       sync.Mutex
	router   http.Handler
	cookies  map[string]string
	tasks    scheduler.Scheduler
	stopping bool
}

func (r *Runtime) Start(directory string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.router != nil {
		return "", errors.New("业务引擎已经启动")
	}
	if directory == "" {
		return "", errors.New("未指定软件数据目录")
	}
	if err := migrations.Initialize(directory); err != nil {
		return "", err
	}
	db, err := public.NewSqlite("data/settings.db", "")
	if err != nil {
		return "", err
	}
	_, err = db.Conn.Exec("INSERT INTO settings (key,value,create_time,update_time,active) SELECT 'desktop_password_enabled','1',?,?,1 WHERE NOT EXISTS (SELECT 1 FROM settings WHERE key='desktop_password_enabled')", time.Now().Format(time.DateTime), time.Now().Format(time.DateTime))
	db.Close()
	if err != nil {
		return "", err
	}
	public.ReloadConfig()
	public.InitLogger(public.LogPath)
	r.router = server.NewRouter()
	r.cookies = make(map[string]string)
	r.stopping = false
	public.RestartFunc = func() {
		public.ReloadConfig()
		public.AuthMu.Lock()
		public.LoginKey = public.GenerateUUID()
		public.AuthMu.Unlock()
	}
	// 窗口和退出由 Rust 管理，原接口不再操作操作系统进程。
	public.ShutdownFunc = func() {}
	r.tasks.Start()
	if public.DesktopPasswordEnabled {
		return "/login", nil
	}
	return "/", nil
}

func (r *Runtime) Request(method, path string, headers http.Header, body []byte) (int, http.Header, []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.router == nil || r.stopping {
		return 503, http.Header{"Content-Type": {"application/json"}}, []byte(`{"status":false,"message":"业务引擎正在关闭"}`)
	}
	if !strings.HasPrefix(path, "/v1/") {
		return 404, http.Header{"Content-Type": {"application/json"}}, []byte(`{"status":false,"message":"业务接口不存在"}`)
	}
	request := httptest.NewRequest(method, "http://allinssl.internal"+path, bytes.NewReader(body))
	request.RemoteAddr = "127.0.0.1:0"
	// WebView2 的请求头是小写，使用 Add 转为 Go 的规范名称后再交给原表单解析。
	request.Header = make(http.Header)
	for name, values := range headers {
		for _, value := range values {
			request.Header.Add(name, value)
		}
	}
	request.Header.Del("Cookie")
	request.Header.Del("Accept-Encoding")
	// 会话只在内存中共享给本站窗口，旧 WebView Cookie 不参与验证。
	for name, value := range r.cookies {
		request.AddCookie(&http.Cookie{Name: name, Value: value})
	}
	writer := httptest.NewRecorder()
	r.router.ServeHTTP(writer, request)
	response := writer.Result()
	for _, cookie := range response.Cookies() {
		if cookie.MaxAge < 0 {
			delete(r.cookies, cookie.Name)
			continue
		}
		r.cookies[cookie.Name] = cookie.Value
	}
	writer.Header().Del("Set-Cookie")
	return writer.Code, writer.Header(), writer.Body.Bytes()
}

func (r *Runtime) Stop() {
	r.mu.Lock()
	r.stopping = true
	r.mu.Unlock()
	r.tasks.Stop()
	workflow.DesktopExecutions.Wait()
	r.mu.Lock()
	defer r.mu.Unlock()
	r.router = nil
	r.cookies = nil
	public.CloseLogger()
}

// ResetPassword 仅由 Rust 完成当前 Windows 用户身份验证之后直接调用。
func (r *Runtime) ResetPassword(password string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.router == nil || r.stopping {
		return errors.New("业务引擎未运行，密码未修改")
	}
	decoded, err := hex.DecodeString(password)
	if err != nil || len(decoded) != md5.Size {
		return errors.New("软件密码摘要无效")
	}
	public.AuthMu.Lock()
	defer public.AuthMu.Unlock()
	db, err := public.NewSqlite("data/settings.db", "")
	if err != nil {
		return err
	}
	defer db.Close()
	var salt string
	if err = db.Conn.QueryRow("SELECT salt FROM users WHERE id=1").Scan(&salt); err != nil {
		return err
	}
	hash := md5.Sum([]byte(password + salt))
	if _, err = db.Conn.Exec("UPDATE users SET password=? WHERE id=1", hex.EncodeToString(hash[:])); err != nil {
		return err
	}
	public.LoginKey = public.GenerateUUID()
	return nil
}
