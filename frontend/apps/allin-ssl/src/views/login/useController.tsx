import md5 from 'crypto-js/md5'
import type { FormInst } from 'naive-ui'

export const useController = () => {
 const loading = ref(false)
 const error = ref('')
 const setupPassword = ref(false)
 const resetting = ref(false)
 const codeImg = ref('')
 const mustCode = ref(false)
 const formRef = ref<FormInst | null>(null)
 const loginData = ref({ password: '', confirm: '', code: '' })

 const handleGetCode = async () => {
  const response = await fetch('/v1/login/get_code')
  codeImg.value = (await response.json()).data
 }
 const handleSubmit = async (event: Event) => {
  event.preventDefault()
  if (loading.value) return
  error.value = ''
  try {
   await formRef.value?.validate()
   if ((setupPassword.value || resetting.value) && loginData.value.password !== loginData.value.confirm) {
    error.value = '两次输入的密码不一致'
    return
   }
   loading.value = true
   const password = md5(`${loginData.value.password}_bt_all_in_ssl`).toString()
   if (resetting.value) {
    const native = (window as Window & { __TAURI_INTERNALS__: { invoke: (command: string, args: object) => Promise<void> } }).__TAURI_INTERNALS__
    await native.invoke('reset_password', { password })
    resetting.value = false
    setupPassword.value = false
    loginData.value = { password: '', confirm: '', code: '' }
    mustCode.value = false
    error.value = '密码已重置，请使用新密码解锁'
    return
   }
   const response = await fetch('/v1/login/sign', {
    method: 'POST', body: new URLSearchParams({ password, code: loginData.value.code, setup: String(setupPassword.value) }),
   })
   const result = await response.json()
   if (!result.status) throw new Error(result.message)
   loginData.value.password = ''
   location.replace('/')
  } catch (cause) {
   error.value = cause instanceof Error ? cause.message : String(cause)
   const state = await (await fetch('/v1/login/state')).json()
   mustCode.value = state.data.must_code
   if (mustCode.value && !resetting.value) await handleGetCode()
  } finally {
   loading.value = false
  }
 }
 const handleKeyup = (event: KeyboardEvent) => {
  if (event.key === 'Enter') void handleSubmit(event)
 }
 const handleForgotPassword = () => {
  resetting.value = !resetting.value
  error.value = ''
  loginData.value = { password: '', confirm: '', code: '' }
 }
 const handleSkipPassword = async () => {
  if (loading.value) return
  loading.value = true
  try {
   const response = await fetch('/v1/login/skip_setup', { method: 'POST' })
   const result = await response.json()
   if (!result.status) throw new Error(result.message)
   location.replace('/')
  } catch (cause) {
   error.value = cause instanceof Error ? cause.message : String(cause)
  } finally { loading.value = false }
 }
 onMounted(async () => {
  // 清除旧版本记住的明文密码，保留主题等其他本地存储。
  localStorage.removeItem('loginData')
  localStorage.removeItem('remember-me')
  const response = await fetch('/v1/login/state')
  const result = await response.json()
  if (!result.status) { error.value = result.message; return }
  if (!result.data.password_enabled && new URLSearchParams(location.search).get('setup') !== '1') { location.replace('/'); return }
  setupPassword.value = result.data.setup
  mustCode.value = result.data.must_code
  if (mustCode.value) await handleGetCode()
 })
 return { loading, error, setupPassword, resetting, loginData, codeImg, mustCode, formRef, handleSubmit, handleKeyup, handleGetCode, handleForgotPassword, handleSkipPassword }
}
