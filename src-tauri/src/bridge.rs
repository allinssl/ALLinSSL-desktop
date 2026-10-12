use std::{
    collections::HashMap,
    ffi::{c_char, c_void, CStr, CString},
    sync::Once,
};

unsafe extern "C" {
    fn _rt0_amd64_windows_lib();
    fn AllinSSLControl(input: *const c_char) -> *mut c_char;
    fn AllinSSLRequest(
        input: *const c_char,
        body: *const c_void,
        length: i32,
        output_length: *mut i64,
    ) -> *mut c_void;
    fn AllinSSLFree(pointer: *mut c_void);
}

static START: Once = Once::new();

#[derive(serde::Deserialize)]
pub struct Reply {
    pub status: u16,
    pub headers: HashMap<String, Vec<String>>,
    #[serde(skip)]
    pub body: Vec<u8>,
}

pub fn control(command: serde_json::Value) -> Result<String, String> {
    // MSVC 不执行 MinGW 的 .ctors 段，显式启动静态 Go 运行时一次。
    START.call_once(|| unsafe { _rt0_amd64_windows_lib() });
    let input = CString::new(command.to_string()).map_err(|error| error.to_string())?;
    let pointer = unsafe { AllinSSLControl(input.as_ptr()) };
    let data = unsafe { CStr::from_ptr(pointer).to_bytes().to_vec() };
    unsafe { AllinSSLFree(pointer.cast()) };
    let result: serde_json::Value =
        serde_json::from_slice(&data).map_err(|error| error.to_string())?;
    if result["status"] != true {
        return Err(result["message"]
            .as_str()
            .unwrap_or("业务引擎操作失败")
            .to_string());
    }
    Ok(result["entry"].as_str().unwrap_or("").to_string())
}

pub fn request(
    method: &str,
    path: &str,
    headers: HashMap<String, Vec<String>>,
    body: &[u8],
) -> Result<Reply, String> {
    let input = CString::new(
        serde_json::json!({"method": method, "path": path, "headers": headers}).to_string(),
    )
    .map_err(|error| error.to_string())?;
    let mut length = 0;
    let pointer = unsafe {
        AllinSSLRequest(
            input.as_ptr(),
            body.as_ptr().cast(),
            body.len() as i32,
            &mut length,
        )
    };
    let data =
        unsafe { std::slice::from_raw_parts(pointer.cast::<u8>(), length as usize).to_vec() };
    unsafe { AllinSSLFree(pointer) };
    let position = data
        .iter()
        .position(|byte| *byte == b'\n')
        .ok_or("业务引擎响应格式错误")?;
    let mut reply: Reply =
        serde_json::from_slice(&data[..position]).map_err(|error| error.to_string())?;
    reply.body = data[position + 1..].to_vec();
    Ok(reply)
}
