; 卸载保留证书、授权和工作流数据。
!macro NSIS_HOOK_PREUNINSTALL
  StrCpy $DeleteAppDataCheckboxState 0
  DetailPrint "卸载后保留程序目录或自选数据目录中的证书、授权和工作流。"
!macroend
