#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use std::{
    collections::HashMap,
    sync::atomic::{AtomicBool, Ordering},
    sync::Mutex,
};
use tauri::{
    menu::{Menu, MenuItem},
    tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent},
    webview::{DownloadEvent, NewWindowResponse},
    Manager, WebviewUrl, WebviewWindowBuilder, WindowEvent,
};
use tauri_plugin_dialog::{DialogExt, MessageDialogKind};
use tauri_plugin_opener::OpenerExt;

mod bridge;
mod data_directory;
mod password;

#[derive(Default)]
struct Backend {
    quitting: AtomicBool,
    stopped: AtomicBool,
    started: AtomicBool,
    resetting: AtomicBool,
}

fn show_main(app: &tauri::AppHandle) {
    if let Some(window) = app.get_webview_window("main") {
        let _ = window.unminimize();
        let _ = window.show();
        let _ = window.set_focus();
    }
}

fn quit(app: &tauri::AppHandle) {
    if app.state::<Backend>().quitting.swap(true, Ordering::SeqCst) {
        return;
    }
    if let Some(window) = app.get_webview_window("main") {
        let _ = window.set_title("ALLinSSL — 正在等待任务完成并退出");
    }
    let handle = app.clone();
    tauri::async_runtime::spawn_blocking(move || {
        let _ = bridge::control(serde_json::json!({"action":"quit"}));
        handle
            .state::<Backend>()
            .stopped
            .store(true, Ordering::SeqCst);
        handle.exit(0);
    });
}

fn fail(app: &tauri::AppHandle, message: String) {
    let handle = app.clone();
    std::thread::spawn(move || {
        handle
            .dialog()
            .message(format!("软件启动或运行失败：{message}"))
            .title("ALLinSSL 运行失败")
            .kind(MessageDialogKind::Error)
            .blocking_show();
        if handle.state::<Backend>().started.load(Ordering::SeqCst) {
            let _ = bridge::control(serde_json::json!({"action":"quit"}));
        }
        handle
            .state::<Backend>()
            .stopped
            .store(true, Ordering::SeqCst);
        handle.exit(1);
    });
}

// 接口下载由 Rust 保存，不导航到接口地址或浏览器错误页。
fn download(app: &tauri::AppHandle, target: tauri::Url) {
    let handle = app.clone();
    tauri::async_runtime::spawn_blocking(move || {
        let path = target.path().to_string()
            + &target
                .query()
                .map(|query| format!("?{query}"))
                .unwrap_or_default();
        let result = (|| -> Result<(), String> {
            let reply = bridge::request("GET", &path, HashMap::new(), &[])?;
            if reply.status >= 400 {
                let data: serde_json::Value =
                    serde_json::from_slice(&reply.body).map_err(|error| error.to_string())?;
                return Err(data["message"].as_str().unwrap_or("下载失败").to_string());
            }
            let disposition = reply
                .headers
                .get("Content-Disposition")
                .and_then(|values| values.first())
                .ok_or("该接口没有提供下载文件")?;
            let name = if let Some(encoded) = disposition.split("filename*=UTF-8''").nth(1) {
                let url =
                    tauri::Url::parse(&format!("https://allinssl.invalid/?filename={encoded}"))
                        .map_err(|error| error.to_string())?;
                url.query_pairs()
                    .next()
                    .ok_or("下载文件名缺失")?
                    .1
                    .into_owned()
            } else {
                disposition
                    .split("filename=")
                    .nth(1)
                    .ok_or("下载文件名缺失")?
                    .trim_matches('"')
                    .to_string()
            };
            let Some(selected) = handle
                .dialog()
                .file()
                .set_file_name(name)
                .blocking_save_file()
            else {
                return Ok(());
            };
            let path = selected.into_path().map_err(|error| error.to_string())?;
            std::fs::write(path, reply.body).map_err(|error| format!("保存文件失败：{error}"))
        })();
        if let Err(error) = result {
            handle
                .dialog()
                .message(error)
                .title("ALLinSSL 下载失败")
                .kind(MessageDialogKind::Error)
                .show(|_| {});
        }
    });
}

fn open_window(
    app: &tauri::AppHandle,
    label: &str,
    page: String,
) -> tauri::Result<tauri::WebviewWindow> {
    let navigation_app = app.clone();
    let popup_app = app.clone();
    let downloads_app = app.clone();
    WebviewWindowBuilder::new(app, label, WebviewUrl::App(page.into()))
        .title("ALLinSSL")
        .decorations(false)
        .inner_size(1440.0, 900.0)
        .min_inner_size(1000.0, 650.0)
        .center()
        .data_directory(app.state::<Mutex<data_directory::DataDirectory>>().lock().unwrap().current.join("webview"))
        .on_navigation(move |target| {
            let local = target.host_str() == Some("tauri.localhost") || target.scheme() == "tauri";
            if local && target.path().starts_with("/v1/") {
                download(&navigation_app, target.clone());
                return false;
            }
            if local
                || target.as_str().starts_with("blob:http://tauri.localhost/")
                || target.as_str() == "about:blank"
            {
                return true;
            }
            if matches!(target.scheme(), "http" | "https") {
                let _ = navigation_app
                    .opener()
                    .open_url(target.as_str(), None::<&str>);
            }
            false
        })
        .on_new_window(move |target, _| {
            if target.host_str() != Some("tauri.localhost") && target.scheme() != "tauri" {
                if matches!(target.scheme(), "http" | "https") {
                    let _ = popup_app.opener().open_url(target.as_str(), None::<&str>);
                }
                return NewWindowResponse::Deny;
            }
            if target.path().starts_with("/v1/") {
                download(&popup_app, target);
                return NewWindowResponse::Deny;
            }
            let handle = popup_app.clone();
            tauri::async_runtime::spawn(async move {
                let label = format!(
                    "page-{}",
                    std::time::SystemTime::now()
                        .duration_since(std::time::UNIX_EPOCH)
                        .unwrap()
                        .as_nanos()
                );
                let page = target.path().to_string()
                    + &target
                        .query()
                        .map(|query| format!("?{query}"))
                        .unwrap_or_default();
                if let Err(error) = open_window(&handle, &label, page) {
                    fail(&handle, format!("打开站内页面失败：{error}"));
                }
            });
            NewWindowResponse::Deny
        })
        .on_download(move |_, event| {
            match event {
                DownloadEvent::Requested { destination, .. } => {
                    let name = destination
                        .file_name()
                        .map(|name| name.to_string_lossy().into_owned())
                        .unwrap_or_default();
                    let selected = downloads_app
                        .dialog()
                        .file()
                        .set_file_name(name)
                        .blocking_save_file();
                    match selected.and_then(|path| path.into_path().ok()) {
                        Some(path) => *destination = path,
                        None => return false,
                    }
                }
                DownloadEvent::Finished { success: false, .. } => {
                    downloads_app
                        .dialog()
                        .message("下载失败，请检查保存路径。")
                        .title("ALLinSSL")
                        .kind(MessageDialogKind::Error)
                        .show(|_| {});
                }
                _ => (),
            }
            true
        })
        .build()
}

#[tauri::command]
async fn reset_password(
    window: tauri::WebviewWindow,
    app: tauri::AppHandle,
    password: String,
) -> Result<(), String> {
    let url = window.url().map_err(|error| error.to_string())?;
    if window.label() != "main"
        || url.path() != "/login"
        || url.host_str() != Some("tauri.localhost")
    {
        return Err("只能从软件解锁页面重置密码".into());
    }
    if password.len() != 32 || !password.bytes().all(|byte| byte.is_ascii_hexdigit()) {
        return Err("软件密码摘要无效".into());
    }
    let hwnd = window.hwnd().map_err(|error| error.to_string())?.0 as usize;
    let backend = app.state::<Backend>();
    if backend.resetting.swap(true, Ordering::SeqCst) {
        return Err("正在验证 Windows 身份，请完成当前操作".into());
    }
    let handle = app.clone();
    let result = tauri::async_runtime::spawn_blocking(move || -> Result<(), String> {
        password::verify_current_user(windows::Win32::Foundation::HWND(hwnd as *mut _))?;
        if handle.state::<Backend>().quitting.load(Ordering::SeqCst) {
            return Err("软件正在退出，密码未修改".into());
        }
        bridge::control(serde_json::json!({"action":"reset_password","password":password}))?;
        Ok(())
    })
    .await
    .map_err(|error| error.to_string());
    backend.resetting.store(false, Ordering::SeqCst);
    result?
}

#[tauri::command]
async fn data_directory(
    window: tauri::WebviewWindow,
    app: tauri::AppHandle,
    select: bool,
) -> Result<data_directory::DataDirectory, String> {
    let url = window.url().map_err(|error| error.to_string())?;
    if window.label() != "main" || url.host_str() != Some("tauri.localhost") || url.path() != "/settings" {
        return Err("只能从软件设置页面调整数据目录".into());
    }
    tauri::async_runtime::spawn_blocking(move || {
        let reply = bridge::request("POST", "/v1/setting/get_setting", HashMap::new(), &[])?;
        if reply.status != 200 {
            return Err("请先解锁软件后调整数据目录".into());
        }
        let state = app.state::<Mutex<data_directory::DataDirectory>>();
        let mut directory = state.lock().map_err(|error| error.to_string())?;
        if !select {
            return Ok(directory.clone());
        }
        let Some(chosen) = app.dialog().file().set_title("选择数据保存目录").set_directory(&directory.selected).blocking_pick_folder() else {
            return Ok(directory.clone());
        };
        let path = chosen.into_path().map_err(|error| error.to_string())?;
        let reply = bridge::request("POST", "/v1/setting/get_setting", HashMap::new(), &[])?;
        if reply.status != 200 || app.state::<Backend>().quitting.load(Ordering::SeqCst) {
            return Err("软件已锁定或正在退出，数据目录未修改".into());
        }
        let temporary = directory.config.with_extension("json.tmp");
        std::fs::write(&temporary, serde_json::json!({"data_dir":path}).to_string())
            .map_err(|error| format!("保存数据目录配置失败：{error}"))?;
        std::fs::rename(&temporary, &directory.config)
            .map_err(|error| format!("保存数据目录配置失败：{error}"))?;
        directory.selected = path;
        Ok(directory.clone())
    }).await.map_err(|error| error.to_string())?
}

fn main() {
    let app = tauri::Builder::default().invoke_handler(tauri::generate_handler![reset_password, data_directory])
        .manage(Backend::default())
        .plugin(tauri_plugin_single_instance::init(|app, _, _| show_main(app)))
        .plugin(tauri_plugin_dialog::init()).plugin(tauri_plugin_opener::init())
        .register_asynchronous_uri_scheme_protocol("tauri", |context, request, responder| {
            let handle = context.app_handle().clone();
            tauri::async_runtime::spawn_blocking(move || {
                let path = request.uri().path().to_string();
                if !path.starts_with("/v1/") {
                    let asset = handle.asset_resolver().get(path).or_else(|| handle.asset_resolver().get("index.html".into()));
                    let response = match asset {
                        Some(asset) => tauri::http::Response::builder().header("Content-Type", asset.mime_type).body(asset.bytes).unwrap(),
                        None => tauri::http::Response::builder().header("Content-Type", "text/html; charset=utf-8").body("<html lang=\"zh-CN\"><body><h2>ALLinSSL 页面资源加载失败</h2><p>请重新安装软件，用户数据不会删除。</p></body></html>".as_bytes().to_vec()).unwrap(),
                    };
                    responder.respond(response);
                    return;
                }
                let query = request.uri().query().map(|query| format!("?{query}")).unwrap_or_default();
                let headers = request.headers().iter().map(|(name, value)| (name.to_string(), vec![value.to_str().unwrap_or_default().to_string()])).collect();
                let reply = bridge::request(request.method().as_str(), &(path.clone() + &query), headers, request.body());
                let response = match reply {
                    Ok(reply) => {
                        let mut response = tauri::http::Response::builder().status(reply.status).header("Cache-Control", "no-store");
                        for (name, values) in reply.headers { for value in values { response = response.header(&name, value); } }
                        response.body(reply.body).unwrap()
                    }
                    Err(error) => tauri::http::Response::builder().status(500).header("Content-Type", "application/json").body(serde_json::json!({"status":false,"message":format!("软件业务调用失败：{error}")}).to_string().into_bytes()).unwrap(),
                };
                let exiting = path == "/v1/setting/shutdown" && response.status().is_success();
                responder.respond(response);
                if exiting { std::thread::sleep(std::time::Duration::from_millis(100)); quit(&handle); }
            });
        })
        .setup(|app| {
            let open = MenuItem::with_id(app, "open", "打开", true, None::<&str>)?;
            let exit = MenuItem::with_id(app, "quit", "退出", true, None::<&str>)?;
            let menu = Menu::with_items(app, &[&open, &exit])?;
            TrayIconBuilder::new().icon(app.default_window_icon().unwrap().clone()).tooltip("ALLinSSL").menu(&menu).show_menu_on_left_click(false)
                .on_menu_event(|app, event| match event.id.as_ref() { "open" => show_main(app), "quit" => quit(app), _ => () })
                .on_tray_icon_event(|tray, event| {
                    if matches!(event, TrayIconEvent::Click { button: MouseButton::Left, button_state: MouseButtonState::Up, .. }) { show_main(tray.app_handle()); }
                }).build(app)?;
            let directory = data_directory::DataDirectory::load(&std::env::current_exe()?)
                .map_err(std::io::Error::other)?;
            std::fs::create_dir_all(directory.current.join("plugins"))?;
            let data_path = directory.current.clone();
            app.manage(Mutex::new(directory));
            match bridge::control(serde_json::json!({"action":"start","directory":data_path.to_string_lossy()})) {
                Ok(entry) => { app.state::<Backend>().started.store(true, Ordering::SeqCst); open_window(app.handle(), "main", entry)?; }
                Err(error) => fail(app.handle(), error),
            }
            Ok(())
        })
        .on_window_event(|window, event| {
            if window.label() == "main" { if let WindowEvent::CloseRequested { api, .. } = event { api.prevent_close(); let _ = window.hide(); } }
        }).build(tauri::generate_context!());
    match app {
        Ok(app) => app.run(|handle, event| {
            if let tauri::RunEvent::ExitRequested { api, .. } = event {
                if !handle.state::<Backend>().stopped.load(Ordering::SeqCst) {
                    api.prevent_exit();
                    quit(handle);
                }
            }
        }),
        Err(error) => {
            let message: Vec<u16> = format!("ALLinSSL 初始化失败：{error}")
                .encode_utf16()
                .chain(Some(0))
                .collect();
            let title: Vec<u16> = "ALLinSSL".encode_utf16().chain(Some(0)).collect();
            unsafe {
                let _ = windows::Win32::UI::WindowsAndMessaging::MessageBoxW(
                    None,
                    windows::core::PCWSTR(message.as_ptr()),
                    windows::core::PCWSTR(title.as_ptr()),
                    windows::Win32::UI::WindowsAndMessaging::MB_ICONERROR,
                );
            }
        }
    }
}
