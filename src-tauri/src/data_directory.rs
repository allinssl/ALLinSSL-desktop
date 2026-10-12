use std::path::{Path, PathBuf};

#[derive(Clone, serde::Serialize)]
pub struct DataDirectory {
    pub current: PathBuf,
    pub selected: PathBuf,
    #[serde(skip)]
    pub config: PathBuf,
}

impl DataDirectory {
    pub fn load(executable: &Path) -> Result<Self, String> {
        let parent = executable.parent().ok_or("无法确定程序所在目录")?;
        let config = parent.join("allinssl.json");
        let selected = match std::fs::read(&config) {
            Ok(bytes) => {
                let value: serde_json::Value = serde_json::from_slice(&bytes)
                    .map_err(|error| format!("读取数据目录配置失败：{error}"))?;
                let path = PathBuf::from(value["data_dir"].as_str().ok_or("数据目录配置缺少 data_dir")?);
                if !path.is_absolute() {
                    return Err("数据目录配置必须使用绝对路径".into());
                }
                path
            }
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => parent.to_path_buf(),
            Err(error) => return Err(format!("读取数据目录配置失败：{error}")),
        };
        Ok(Self { current: selected.clone(), selected, config })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn default_and_selected_directory() {
        let root = std::env::temp_dir().join(format!("ALLinSSL 数据目录 {}", std::process::id()));
        std::fs::create_dir_all(&root).unwrap();
        let executable = root.join("ALLinSSL.exe");
        let initial = DataDirectory::load(&executable).unwrap();
        assert_eq!(initial.current, root);
        let selected = root.join("自选 数据");
        std::fs::write(&initial.config, serde_json::json!({"data_dir":selected}).to_string()).unwrap();
        let reopened = DataDirectory::load(&executable).unwrap();
        assert_eq!(reopened.current, selected);
        assert!(!selected.exists(), "读取配置不应提前初始化新目录");
        std::fs::write(&initial.config, "{\"data_dir\":\"relative\"}").unwrap();
        assert!(DataDirectory::load(&executable).is_err());
        std::fs::write(&initial.config, "{").unwrap();
        assert!(DataDirectory::load(&executable).is_err());
        std::fs::remove_dir_all(root).unwrap();
    }
}
