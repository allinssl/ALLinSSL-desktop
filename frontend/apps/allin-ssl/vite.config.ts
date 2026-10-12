import { defineConfig } from 'vitest/config'
import path, { resolve } from 'path' // 路径处理
import Vue from '@vitejs/plugin-vue' // vue处理
import VueJsx from '@vitejs/plugin-vue-jsx' // jsx处理
// import basicSsl from '@vitejs/plugin-basic-ssl' // https协议兼容
import legacy from '@vitejs/plugin-legacy' // 浏览器兼容
import VueDevTools from 'vite-plugin-vue-devtools' // vue3调试工具
import { createSvgIconsPlugin } from 'vite-plugin-svg-icons'
import AutoImport from 'unplugin-auto-import/vite' // 方法自动引入
import { NaiveUiResolver } from 'unplugin-vue-components/resolvers' // 引入naive-ui组件
import Components from 'unplugin-vue-components/vite' // 组件自动引入
// import { compression } from 'vite-plugin-compression2'
// import

const packPath = 'static/' // 打包后的vite目录
const isDev = process.env.NODE_ENV === 'development' // 开发环境
export default defineConfig({
  mode: isDev ? "development" : "production",
  base: "/",
  // 插件配置
  plugins: [
    // vue处理
    Vue({
      script: {
        defineModel: true,
      },
    }),

    // jsx处理
    VueJsx(),
    // 压缩gzip
    // compression(),
    // 方法自动引入
    AutoImport({
      imports: ["vue", "@vueuse/core", "pinia", "vue-router"],
      dts: "./types/auto-imports.d.ts", // 生成的d.ts文件
      eslintrc: {
        enabled: true, // 默认false, true启用。生成一次就可以，避免每次工程启动都生成
        filepath: "./types/.eslintrc-auto-import.json", // 生成json文件
        globalsPropValue: true,
      },
    }),

    // 组件自动引入
    Components({
      dts: "./types/components.d.ts", // 生成的d.ts文件
      dirs: ["src/components"], // 指定扫描的文件夹
      extensions: ["vue", "tsx"], // 指定扫描的文件类型
      directoryAsNamespace: false, // 是否将目录作为命名空间
      resolvers: [NaiveUiResolver()],
    }),

    // https协议兼容
    // basicSsl(),

    // 浏览器兼容
    isDev &&
      legacy({
        targets: [">0.1%", "Firefox > 55", "Chrome > 60", "safari > 11"],
      }),

    // vue3调试工具
    isDev && VueDevTools(),

    // 创建 svg 图标
    createSvgIconsPlugin({
      iconDirs: [path.resolve(process.cwd(), "src/assets/icons/svg/")],
      symbolId: "icon-[dir]-[name]",
    }),
  ],
  resolve: {
    // 别名配置
    alias: {
      "@layout": path.resolve(__dirname, "src/views/layout"),
      "@login": path.resolve(__dirname, "src/views/login"),
      "@404": path.resolve(__dirname, "src/views/404"),
      "@certManage": path.resolve(__dirname, "src/views/certManage"),
      "@certApply": path.resolve(__dirname, "src/views/certApply"),
      "@autoDeploy": path.resolve(__dirname, "src/views/autoDeploy"),
      "@workflowView": path.resolve(
        __dirname,
        "src/views/autoDeploy/children/workflowView"
      ),
      "@autoApiManage": path.resolve(__dirname, "src/views/autoApiManage"),
      "@home": path.resolve(__dirname, "src/views/home"),
      "@monitor": path.resolve(__dirname, "src/views/monitor"),
      "@settings": path.resolve(__dirname, "src/views/settings"),
      "@test": path.resolve(__dirname, "src/views/test"),
      "@api": path.resolve(__dirname, "src/api"),
      "@assets": path.resolve(__dirname, "src/assets"),
      "@components": path.resolve(__dirname, "src/components"),
      "@public": path.resolve(__dirname, "src/public"),
      "@router": path.resolve(__dirname, "src/router"),
      "@locales": path.resolve(__dirname, "src/locales"),
      "@config": path.resolve(__dirname, "src/config"),
      "@styles": path.resolve(__dirname, "src/styles"),
      "@types": path.resolve(__dirname, "src/types"),
      "@lib": path.resolve(__dirname, "src/lib"),
      "@": path.resolve(__dirname, "src"),
    },
  },
  build: {
    outDir: resolve(__dirname, "../../../static/build"),
    emptyOutDir: true,
    minify: "terser", // 混淆器，terser构建后文件体积更小
    // assetsDir: `${packPath}/`, // 静态资源目录
    sourcemap: false,
    cssCodeSplit: false, // 不分割css代码
    reportCompressedSize: false, // 不统计gzip压缩后的文件大小
    chunkSizeWarningLimit: 800, // 警告阈值
    assetsInlineLimit: 2048, // 小于2kb的资源内联
    modulePreload: false, // 禁用预加载
    terserOptions: {
      // 打包后移除console和注释
      compress: {
        drop_console: !isDev, // 生产环境移除console
        drop_debugger: !isDev, // 生产环境移除debugger
      },
    },
    rollupOptions: {
      input: {
        main: resolve(__dirname, "index.html"), // 主页面
      },
      strictDeprecations: true, // 严格弃用
      output: {
        entryFileNames: `${packPath}js/[name]-[hash].js`,
        chunkFileNames: `${packPath}js/[name]-[hash].js`,
        assetFileNames: (chunkInfo) => {
          const { names } = chunkInfo;
          let ext = "[ext]";
          if (names && names.length > 0) {
            const name = names[0];
            const str = name.substring(name.lastIndexOf(".") + 1);
            if (str === "ttf" || str === "woff" || str === "woff2")
              ext = "font";
          }
          return `${packPath}${ext}/[name]-[hash].[ext]`;
        },
      },
    },
  },
  server: {
    // https: { rejectUnauthorized: false },
    host: "0.0.0.0",
    port: 5173,
    proxy: {
      "/api": {
        // target: `http://${'192.168.168.25'}:${37628}`,
        // target: `http://${'192.168.168.121'}:${33488}`,
        target: `http://${"192.168.168.64"}:${20773}`,
        changeOrigin: true, // 是否改变源
        rewrite: (path: string) => path.replace(/^\/api/, ""), // 重写路径
        secure: false, // 如果是https接口，需要配置这个参数
        ws: false, // 是否启用websocket
      },
    },
  },
  test: {
    include: ["src/**/*.spec.ts"],
  },
});

