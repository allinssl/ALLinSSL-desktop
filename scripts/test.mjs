import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('../', import.meta.url));
const result = spawnSync(process.env.GO_BINARY || 'go', ['test', './backend/desktop', './backend/internal/cert/deploy/plugin'], {
  cwd: root, stdio: 'inherit', env: { ...process.env, CGO_ENABLED: '0' },
});
if (result.error) throw result.error;
process.exit(result.status || 0);
