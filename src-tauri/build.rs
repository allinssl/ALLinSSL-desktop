fn main() {
    println!("cargo:rustc-link-search=native=native");
    println!("cargo:rustc-link-lib=static=allinssl");
    println!("cargo:rustc-link-lib=static=legacy_stdio_definitions");
    println!("cargo:rerun-if-changed=native/allinssl.lib");
    let attributes = tauri_build::Attributes::new()
        .app_manifest(tauri_build::AppManifest::new().commands(&["reset_password", "data_directory"]));
    tauri_build::try_build(attributes).expect("桌面工程构建配置失败");
}
