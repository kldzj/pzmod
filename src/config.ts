import fs from 'fs/promises';
import { existsSync } from 'fs';
import { homedir } from 'os';
import { resolve } from 'path';

export const configPath = resolve(homedir(), '.pzmod.json');

export interface Config {
  apiKey?: string;
}

export async function loadConfig(): Promise<Config> {
  if (!existsSync(configPath)) {
    return {};
  }

  return JSON.parse(await fs.readFile(configPath, 'utf8'));
}

export async function saveConfig(config: Config): Promise<Config> {
  await fs.writeFile(configPath, JSON.stringify(config, null, 2), 'utf8');
  return config;
}
