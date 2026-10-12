package public

import (
	"strconv"
	"sync"
)

var AuthMu sync.RWMutex
var DesktopPasswordEnabled = true

var SessionKey, LogPath string
var LoginKey = GenerateUUID()
var TimeOut = 3600
var ShutdownFunc func()
var RestartFunc func()

func ReloadConfig() {
	AuthMu.Lock()
	defer AuthMu.Unlock()
	DesktopPasswordEnabled = GetSettingIgnoreError("desktop_password_enabled") != "0"
	SessionKey = GetSettingIgnoreError("session_key")
	LogPath = GetSettingIgnoreError("log_path")

	settingStr := GetSettingIgnoreError("timeout")
	setting, err := strconv.Atoi(settingStr)
	if err != nil {
		TimeOut = 3600
	} else {
		TimeOut = setting
	}

}

// OpLog 操作日志
type OpLog struct {
	OpType   string `db:"op_type"`
	OpUser   string `db:"op_user"`
	OpTime   string `db:"op_time"`
	OpDetail string `db:"op_detail"`
	OpResult string `db:"op_result"`
	IP       string `db:"ip"`
}
