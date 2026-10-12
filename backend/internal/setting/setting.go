package setting

import (
	"ALLinSSL/backend/public"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"
)

type Setting struct {
	Timeout         int    `json:"timeout" form:"timeout"`
	PluginPath      string `json:"plugin_path" form:"plugin_path"`
	PasswordEnabled *bool  `json:"password_enabled,omitempty" form:"password_enabled"`
	CurrentPassword string `json:"-" form:"current_password"`
}

func Get() (Setting, error) {
	public.AuthMu.RLock()
	defer public.AuthMu.RUnlock()
	enabled := public.DesktopPasswordEnabled
	return Setting{Timeout: public.TimeOut, PluginPath: public.GetSettingIgnoreError("plugin_dir"), PasswordEnabled: &enabled}, nil
}

func Save(setting *Setting) error {
	if setting.Timeout <= 0 {
		return fmt.Errorf("超时时间必须大于零")
	}
	public.AuthMu.Lock()
	defer public.AuthMu.Unlock()
	db, err := public.NewSqlite("data/settings.db", "settings")
	if err != nil {
		return err
	}
	defer db.Close()
	if setting.PasswordEnabled != nil && *setting.PasswordEnabled != public.DesktopPasswordEnabled {
		if *setting.PasswordEnabled {
			return fmt.Errorf("请先设置软件密码，再启用密码保护")
		}
		var password, salt string
		if err = db.Conn.QueryRow("SELECT password,salt FROM users WHERE id=1").Scan(&password, &salt); err != nil {
			return err
		}
		hash := md5.Sum([]byte(setting.CurrentPassword + salt))
		if setting.CurrentPassword == "" || hex.EncodeToString(hash[:]) != password {
			return fmt.Errorf("软件密码验证失败，密码保护未关闭")
		}
	}
	_, err = db.Conn.Exec("UPDATE settings SET value=? WHERE key='timeout'", setting.Timeout)
	if err != nil {
		return err
	}
	public.TimeOut = setting.Timeout
	if setting.PasswordEnabled != nil && !*setting.PasswordEnabled && public.DesktopPasswordEnabled {
		if err = public.UpdateSetting("desktop_password_enabled", "0"); err != nil {
			return err
		}
		public.DesktopPasswordEnabled = false
		public.LoginKey = public.GenerateUUID()
	}
	if setting.PluginPath != "" && setting.PluginPath != public.GetSettingIgnoreError("plugin_dir") {
		return public.UpdateSetting("plugin_dir", setting.PluginPath)
	}
	return nil
}

func Shutdown() {
	go func() {
		time.Sleep(time.Millisecond * 100)
		public.ShutdownFunc()
	}()
	return
}

func Restart() {
	public.RestartFunc()
}

// 桌面版展示自身版本，不向原网页服务查询更新。
func GetVersion() (map[string]string, error) {
	return map[string]string{"version": "v0.0.1", "new_version": "v0.0.1", "update": "0", "log": "", "date": ""}, nil
}
