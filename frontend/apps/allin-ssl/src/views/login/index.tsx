// External Libraries
import { NInput, NButton, NForm, NFormItem, NIcon, NImage } from 'naive-ui'
import { LockOutlined, CodeOutlined } from '@vicons/antd'

// Absolute Internal Imports
import { useTheme, useThemeCssVar } from '@baota/naive-ui/theme'
import { $t } from '@locales/index'

// Relative Internal Imports
import { useController } from './useController'

// Side-effect Imports
import styles from './index.module.css'

export default defineComponent({
	name: 'LoginView',
	setup() {
		const { loading, error, setupPassword, resetting, handleSkipPassword, handleForgotPassword, handleSubmit, handleKeyup, loginData, handleGetCode, codeImg, mustCode, formRef } =
			useController()
		const { isDark } = useTheme()
		const cssVar = useThemeCssVar(['textColor2', 'actionColor', 'errorColor', 'primaryColor', 'primaryColorSuppl'])

		return () => (
			<div style={cssVar.value}>
				<div
					class={styles.container}
					style={`background-image:${isDark.value ? 'url(/static/images/login-bg-dark.svg)' : 'url(/static/images/login-bg.svg)'};`}
				>
					<div class={styles.loginBox}>
						<div class={styles.leftSection}>
							<h2 class={styles.leftTitle}>
								<img src="/static/images/logo.png" alt="logo" class={styles.logo} />
								<span>{$t('t_2_1747047214975')}</span>
							</h2>
							<div class={styles.leftImageWrapper}>
								<img src="/static/images/login-display.svg" alt={$t('t_1_1744164835667')} class={styles.leftImage} />
							</div>
						</div>
						<div class={styles.rightSection}>
							<div class={styles.formContainer}>
								<h1 class={styles.title}>{setupPassword.value ? "设置软件密码" : resetting.value ? "重置软件密码" : "解锁 ALLinSSL"}</h1>
								<NForm ref={formRef} model={loginData.value} onSubmit={handleSubmit} class={styles.formWrapper}>
									<div class={styles.formContent}>
										<div class={styles.formInputs}>
											<NFormItem
												show-label={false}
												path="password"
												rule={{ required: true, message: $t('t_4_1744164840458'), trigger: ['input', 'blur'] }}
											>
												<NInput
													onKeyup={handleKeyup}
													disabled={loading.value}
													v-model:value={loginData.value.password}
													type="password"
													placeholder={$t('t_4_1744164840458')}
													clearable
													size="large"
													showPasswordOn="click"
												>
													{{
														prefix: () => <NIcon component={LockOutlined} class={styles.icon} />,
													}}
												</NInput>
											</NFormItem>
											{(setupPassword.value || resetting.value) && (
            <NFormItem show-label={false} path="confirm" rule={{ required: true, message: "请再次输入软件密码", trigger: ['input', 'blur'] }}>
             <NInput v-model:value={loginData.value.confirm} onKeyup={handleKeyup} disabled={loading.value} type="password" placeholder="再次输入软件密码" size="large" showPasswordOn="click" />
            </NFormItem>
           )}
           {mustCode.value && !resetting.value ? (
												<NFormItem
													show-label={false}
													path="code"
													rule={{
														required: mustCode.value,
														message: $t('t_25_1745289355721'),
														trigger: ['input', 'blur']
													}}
												>
													<NInput
														onKeyup={handleKeyup}
														disabled={loading.value}
														v-model:value={loginData.value.code}
														type="text"
														placeholder={$t('t_25_1745289355721')}
														clearable
														size="large"
													>
														{{
															prefix: () => <NIcon component={CodeOutlined} class={styles.icon} />,
															suffix: () => (
																<span
																	onClick={handleGetCode}
																	title={$t('t_0_1745936396853')}
																	class={`w-[10rem] h-[4rem] mr-[-1.5rem] flex items-center justify-center relative z-[999] cursor-pointer bg-slate-400 rounded-r-[6px] ${styles.codeImageContainer}`}
																>
																	<NImage
																		src={codeImg.value}
																		preview-disabled
																		class="max-w-full max-h-full object-contain"
																	/>
																</span>
															),
														}}
													</NInput>
												</NFormItem>
											) : null}
										</div>

										<div class={styles.formActions}>
           {!setupPassword.value && <div class={styles.rememberSection}>
            <NButton text disabled={loading.value} onClick={handleForgotPassword}>{resetting.value ? "返回解锁" : "忘记密码"}</NButton>
           </div>}
           {resetting.value && <p>通过当前 Windows 用户的系统密码或 Windows Hello/PIN 验证后，保存新软件密码。</p>}

											{error.value && <div class={styles.error}>{error.value}</div>}
											<NButton
												type="primary"
												size="large"
												block
												loading={loading.value}
												attrType="submit"
												onClick={handleSubmit}
											>
												{loading.value ? "正在处理…" : setupPassword.value ? "设置密码并解锁" : resetting.value ? "验证 Windows 身份并重置" : "解锁"}
											</NButton>
           {setupPassword.value && <NButton block text disabled={loading.value} onClick={handleSkipPassword}>暂不设置密码</NButton>}
										</div>
									</div>
								</NForm>
							</div>
						</div>
					</div>
				</div>
			</div>
		)
	},
})
