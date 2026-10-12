import { NCard, NButton, NTooltip, NInput, NFormItem } from "naive-ui";
import { useMessage } from "@baota/naive-ui/hooks";
import { $t } from "@locales/index";
import { useStore } from "../useStore";
import { useController, useGeneralSettingsController } from "../useController";
import styles from "./index.module.css";

/**
 * 常用设置标签页组件
 */
export default defineComponent({
  name: "GeneralSettings",
  setup() {
    const { generalSettings } = useStore();
    const { handleSaveGeneralSettings, handleDownloadData } = useController();
    const { GeneralForm } = useGeneralSettingsController();
    const directory = ref({ current: '', selected: '' });
    const choosing = ref(false);
    const message = useMessage();
    const native = (window as Window & { __TAURI_INTERNALS__: { invoke: (command: string, args: object) => Promise<{current: string; selected: string}> } }).__TAURI_INTERNALS__;
    onMounted(async () => {
      try {
        directory.value = await native.invoke('data_directory', { select: false });
      } catch (error) {
        message.error(String(error));
      }
    });
    return () => (
      <div class={`${styles.generalSettingsCard} flex flex-col`}>
        <NCard class={styles.baseCard}>
          <GeneralForm labelPlacement="top" class={styles.formGrid} />
          <NFormItem label="数据保存目录" labelPlacement="top">
            <div class="w-full">
              <div class="flex gap-4">
                <NInput value={directory.value.selected} readonly />
                <NButton loading={choosing.value} onClick={async () => {
                  choosing.value = true;
                  try {
                    const selected = await native.invoke('data_directory', { select: true });
                    if (selected.selected !== directory.value.selected) message.success('目录已保存，退出并重新启动软件后生效');
                    directory.value = selected;
                  } catch (error) {
                    message.error(String(error));
                  } finally {
                    choosing.value = false;
                  }
                }}>更改目录</NButton>
              </div>
              <p class="mt-3 text-gray-500">修改后重启生效，原目录数据保留在原处，不自动迁移。</p>
              {directory.value.selected !== directory.value.current && <p class="text-gray-500">当前使用：{directory.value.current}</p>}
            </div>
          </NFormItem>
        </NCard>
        <div class="mt-[1rem] flex">
          <NButton
            type="primary"
            class="gradient-primary-btn mr-[2rem]"
            onClick={() => handleSaveGeneralSettings(generalSettings.value)}
          >
            {$t("t_9_1745464078110")}
          </NButton>
          <NTooltip
            v-slots={{
              trigger: () => (
                <NButton
                  class="gradient-primary-btn"
                  type="primary"
                  onClick={handleDownloadData}
                >
                  下载数据
                </NButton>
              ),
            }}
          >
            下载工作流、通知、证书、api授权数据，可直接将数据库文件复制到allinssl的data下使用
          </NTooltip>
        </div>
      </div>
    );
  },
});
