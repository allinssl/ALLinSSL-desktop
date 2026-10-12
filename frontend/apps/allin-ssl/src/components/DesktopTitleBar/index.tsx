import { defineComponent, ref, onMounted, onUnmounted } from 'vue'
import { NLayoutHeader } from 'naive-ui'
import { useMessage } from '@baota/naive-ui/hooks'
import { useTheme, useThemeCssVar } from '@baota/naive-ui/theme'
import styles from './index.module.css'

export default defineComponent({
  name: 'DesktopTitleBar',
  setup(_, { slots }) {
    const { isDark } = useTheme()
    const cssVars = useThemeCssVar(['textColor1', 'actionColor', 'borderColor', 'primaryColor'])
    const maximized = ref(false)
    const message = useMessage()
    const native = (window as Window & { __TAURI_INTERNALS__: {
      invoke: (command: string) => Promise<boolean | void>
    } }).__TAURI_INTERNALS__

    const handleAction = async (action: 'minimize' | 'toggle_maximize' | 'close' | 'is_maximized') => {
      try {
        const result = await native.invoke(`plugin:window|${action}`)
        if (action === 'is_maximized') maximized.value = result as boolean
      } catch (error) {
        message.error(`窗口操作失败：${String(error)}`)
      }
    }
    const updateMaximized = () => { void handleAction('is_maximized') }
    onMounted(() => {
      updateMaximized()
      window.addEventListener('resize', updateMaximized)
    })
    onUnmounted(() => window.removeEventListener('resize', updateMaximized))

    return () => (
      <NLayoutHeader class={styles.titlebar} style={cssVars.value} data-tauri-drag-region="deep">
        <div class={styles.left}>
          {slots.left ? slots.left() : (
            <div class={styles.brand}>
              <img src={`/static/images/logo${isDark.value ? '-dark' : ''}.png`} alt="ALLinSSL" />
              <span>ALLinSSL</span>
            </div>
          )}
        </div>
        <div class={styles.dragSpace} data-tauri-drag-region="deep" />
        <div class={styles.right} data-tauri-drag-region="false">{slots.right?.()}</div>
        <div class={styles.controls} data-tauri-drag-region="false">
          <button class={styles.button} type="button" aria-label="最小化" title="最小化" onClick={() => handleAction('minimize')}>
            <svg width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor"><path d="M1 6.5h10" /></svg>
          </button>
          <button class={styles.button} type="button" aria-label={maximized.value ? '还原' : '最大化'} title={maximized.value ? '还原' : '最大化'} onClick={() => handleAction('toggle_maximize')}>
            <svg width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor">
              {maximized.value ? <path d="M3.5 3.5v-2h7v7h-2M1.5 3.5h7v7h-7z" /> : <rect x="1.5" y="1.5" width="9" height="9" />}
            </svg>
          </button>
          <button class={[styles.button, styles.close].join(' ')} type="button" aria-label="关闭窗口" title="关闭窗口" onClick={() => handleAction('close')}>
            <svg width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor"><path d="m1.5 1.5 9 9m0-9-9 9" /></svg>
          </button>
        </div>
      </NLayoutHeader>
    )
  },
})
