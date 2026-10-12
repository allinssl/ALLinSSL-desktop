use windows::{
    core::{w, HSTRING, PCWSTR, PWSTR},
    Security::Credentials::UI::{
        UserConsentVerificationResult, UserConsentVerifier, UserConsentVerifierAvailability,
    },
    Win32::{
        Foundation::{CloseHandle, HANDLE, HWND},
        Security::{
            Credentials::{
                CredUIPromptForWindowsCredentialsW, CredUnPackAuthenticationBufferW,
                CREDUIWIN_ENUMERATE_CURRENT_USER, CREDUIWIN_GENERIC, CREDUI_INFOW, CRED_PACK_FLAGS,
            },
            EqualSid, GetTokenInformation, LogonUserW, TokenUser, LOGON32_LOGON_INTERACTIVE,
            LOGON32_PROVIDER_DEFAULT, TOKEN_QUERY, TOKEN_USER,
        },
        System::{
            Com::CoTaskMemFree,
            Threading::{GetCurrentProcess, OpenProcessToken},
            WinRT::{
                IUserConsentVerifierInterop, RoInitialize, RoUninitialize, RO_INIT_MULTITHREADED,
            },
        },
    },
};

// 所有系统凭据只在 Rust 与 Windows 之间交换，不保存或传递给业务页面。
pub fn verify_current_user(hwnd: HWND) -> Result<(), String> {
    let initialized = unsafe { RoInitialize(RO_INIT_MULTITHREADED) }.is_ok();
    let availability =
        UserConsentVerifier::CheckAvailabilityAsync().and_then(|operation| operation.join());
    if availability == Ok(UserConsentVerifierAvailability::Available) {
        let result = (|| -> windows::core::Result<UserConsentVerificationResult> {
            let interop: IUserConsentVerifierInterop =
                windows::core::factory::<UserConsentVerifier, IUserConsentVerifierInterop>()?;
            let operation: windows_future::IAsyncOperation<UserConsentVerificationResult> = unsafe {
                interop.RequestVerificationForWindowAsync(
                    hwnd,
                    &HSTRING::from("验证 Windows 身份以重置 ALLinSSL 软件密码"),
                )?
            };
            operation.join()
        })();
        if initialized {
            unsafe { RoUninitialize() };
        }
        return match result {
            Ok(UserConsentVerificationResult::Verified) => Ok(()),
            Ok(UserConsentVerificationResult::Canceled) => {
                Err("已取消 Windows 身份验证，密码未修改".into())
            }
            Ok(_) => Err("Windows 身份验证未通过，密码未修改".into()),
            Err(error) => Err(format!("Windows 身份验证失败：{error}")),
        };
    }
    if initialized {
        unsafe { RoUninitialize() };
    }

    // 没有配置 Hello/PIN 的系统通过 Windows 原生凭据窗口验证账户密码。
    let info = CREDUI_INFOW {
        cbSize: std::mem::size_of::<CREDUI_INFOW>() as u32,
        hwndParent: hwnd,
        pszMessageText: w!("输入当前 Windows 账户的系统密码，以重置 ALLinSSL 软件密码。"),
        pszCaptionText: w!("ALLinSSL — 验证 Windows 身份"),
        ..Default::default()
    };
    let mut package = 0;
    let mut buffer = std::ptr::null_mut();
    let mut size = 0;
    let status = unsafe {
        CredUIPromptForWindowsCredentialsW(
            Some(&info),
            0,
            &mut package,
            None,
            0,
            &mut buffer,
            &mut size,
            None,
            CREDUIWIN_GENERIC | CREDUIWIN_ENUMERATE_CURRENT_USER,
        )
    };
    if status == 1223 {
        return Err("已取消 Windows 身份验证，密码未修改".into());
    }
    if status != 0 {
        return Err(format!("无法打开 Windows 身份验证窗口（错误码 {status}）"));
    }
    let mut username = [0u16; 514];
    let mut domain = [0u16; 256];
    let mut password = [0u16; 514];
    let mut username_size = username.len() as u32;
    let mut domain_size = domain.len() as u32;
    let mut password_size = password.len() as u32;
    let unpacked = unsafe {
        CredUnPackAuthenticationBufferW(
            CRED_PACK_FLAGS(0),
            buffer,
            size,
            Some(PWSTR(username.as_mut_ptr())),
            &mut username_size,
            Some(PWSTR(domain.as_mut_ptr())),
            Some(&mut domain_size),
            Some(PWSTR(password.as_mut_ptr())),
            &mut password_size,
        )
    };
    unsafe {
        for byte in std::slice::from_raw_parts_mut(buffer.cast::<u8>(), size as usize) {
            std::ptr::write_volatile(byte, 0);
        }
        CoTaskMemFree(Some(buffer));
    }
    let mut token = HANDLE::default();
    let verified = unpacked.and_then(|_| unsafe {
        LogonUserW(
            PCWSTR(username.as_ptr()),
            if domain[0] == 0 {
                PCWSTR::null()
            } else {
                PCWSTR(domain.as_ptr())
            },
            PCWSTR(password.as_ptr()),
            LOGON32_LOGON_INTERACTIVE,
            LOGON32_PROVIDER_DEFAULT,
            &mut token,
        )
    });
    for value in &mut password {
        unsafe { std::ptr::write_volatile(value, 0) };
    }
    verified.map_err(|_| "Windows 账户密码验证失败，软件密码未修改".to_string())?;
    let result = check_current_identity(token);
    unsafe {
        let _ = CloseHandle(token);
    }
    result
}

fn check_current_identity(token: HANDLE) -> Result<(), String> {
    let mut current = HANDLE::default();
    unsafe { OpenProcessToken(GetCurrentProcess(), TOKEN_QUERY, &mut current) }
        .map_err(|error| format!("读取当前 Windows 身份失败：{error}"))?;
    let mut candidate = [0usize; 64];
    let mut owner = [0usize; 64];
    let mut size = 0;
    let result = unsafe {
        GetTokenInformation(
            token,
            TokenUser,
            Some(candidate.as_mut_ptr().cast()),
            std::mem::size_of_val(&candidate) as u32,
            &mut size,
        )
    }
    .and_then(|_| unsafe {
        GetTokenInformation(
            current,
            TokenUser,
            Some(owner.as_mut_ptr().cast()),
            std::mem::size_of_val(&owner) as u32,
            &mut size,
        )
    })
    .and_then(|_| unsafe {
        EqualSid(
            (*candidate.as_ptr().cast::<TOKEN_USER>()).User.Sid,
            (*owner.as_ptr().cast::<TOKEN_USER>()).User.Sid,
        )
    });
    unsafe {
        let _ = CloseHandle(current);
    }
    result.map_err(|_| "必须使用运行 ALLinSSL 的当前 Windows 用户完成验证".into())
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn current_user_is_required() {
        let mut token = HANDLE::default();
        unsafe { OpenProcessToken(GetCurrentProcess(), TOKEN_QUERY, &mut token) }.unwrap();
        assert!(
            check_current_identity(token).is_ok(),
            "当前 Windows 用户验证失败"
        );
        unsafe { CloseHandle(token) }.unwrap();
        assert!(
            check_current_identity(HANDLE::default()).is_err(),
            "无效身份不应通过验证"
        );
    }
}
