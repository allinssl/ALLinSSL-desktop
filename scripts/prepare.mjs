import { spawnSync } from 'node:child_process';
import { mkdirSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { join } from 'node:path';

const root = fileURLToPath(new URL('../', import.meta.url));
const go = process.env.GO_BINARY || 'go';
const env = { ...process.env, GOOS: 'windows', GOARCH: 'amd64', NODE_ENV: 'production', NODE_OPTIONS: '--max-old-space-size=8192' };

// Windows 的 pnpm 命令由命令解释器运行，参数仅使用固定的构建命令。
const frontend = spawnSync('pnpm', ['exec', 'turbo', 'run', 'build', '--filter=allin-ssl'], {
  cwd: join(root, 'frontend'), env, stdio: 'inherit', shell: process.platform === 'win32',
});
if (frontend.status !== 0) process.exit(frontend.status || 1);
mkdirSync(join(root, 'src-tauri/native'), { recursive: true });
const result = spawnSync(go, ['build', '-buildmode=c-archive', '-trimpath', '-ldflags=-s -w', '-o', 'src-tauri/native/allinssl.lib', './cmd/native'], {
  cwd: root, env: { ...env, CGO_ENABLED: '1', CC: process.env.GO_NATIVE_CC || process.env.CC || 'gcc' }, stdio: 'inherit',
});
if (result.error) throw result.error;
if (result.status !== 0) process.exit(result.status || 1);
console.log('前端和内置 Go 静态库构建完成；业务后端与两个内置插件不生成独立 exe。');
