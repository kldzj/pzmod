// Pulls the canonical install scripts from the code branch into public/ at build
// time, so pzmod.dev/install.sh and /install.ps1 always mirror the single source
// in the main repo. This runs automatically before `astro build` (npm "prebuild"),
// so the scripts are never committed to (or edited on) the website branch.
//
// In CI a missing script fails the build (never deploy a broken install URL);
// locally it only warns, so you can build/preview before the scripts land on the
// target ref. Override the ref with PZMOD_SCRIPTS_REF.
import { writeFile, mkdir } from 'node:fs/promises';

const BRANCH = process.env.PZMOD_SCRIPTS_REF || 'main';
const RAW = `https://raw.githubusercontent.com/kldzj/pzmod/${BRANCH}`;
const FILES = ['install.sh', 'install.ps1'];
const strict = process.env.CI === 'true' || process.env.PZMOD_FETCH_STRICT === '1';

await mkdir('public', { recursive: true });

for (const name of FILES) {
  const url = `${RAW}/${name}`;
  const res = await fetch(url);
  if (!res.ok) {
    const msg = `fetch-install: ${url} -> HTTP ${res.status}`;
    if (strict) {
      console.error(msg);
      process.exit(1);
    }
    console.warn(`${msg} (skipping; set PZMOD_FETCH_STRICT=1 to fail)`);
    continue;
  }
  const body = await res.text();
  await writeFile(`public/${name}`, body);
  console.log(`fetch-install: wrote public/${name} (${body.length} bytes) from ${BRANCH}`);
}
