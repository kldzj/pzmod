import { EOL, homedir } from 'os';
import fs from 'fs/promises';
import { basename, dirname, join } from 'path';
import chalk from 'chalk';

export interface ServerConfig {
  [key: string]: {
    comments: string[];
    value: string | number | boolean;
  };
}

function parseValue(value: string): string | number | boolean {
  if (value.toLowerCase() === 'true') return true;
  if (value.toLowerCase() === 'false') return false;
  if (value.trim() === '') return '';
  if (!isNaN(Number(value))) return Number(value);
  return value;
}

function isValidServerConfig(config: ServerConfig): boolean {
  return config.hasOwnProperty('PublicName') && config.hasOwnProperty('Mods') && config.hasOwnProperty('WorkshopItems');
}

export async function readServerConfig(path: string, backup: boolean): Promise<ServerConfig> {
  const file = await fs.readFile(path, 'utf8');
  if (!file.length) {
    throw new Error('File is empty');
  }

  if (backup) {
    const backupPath = join(homedir(), `.pzmod_${basename(path)}.bak`);
    await fs.mkdir(dirname(backupPath), { recursive: true });
    await fs.writeFile(backupPath, file);
  }

  const lines = file.split(EOL);
  const config: ServerConfig = {};

  let comments: string[] = [];
  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed.length) continue;
    if (trimmed.startsWith('#')) {
      comments.push(line);
      continue;
    }

    const [key, ...value] = trimmed.split('=');
    config[key] = {
      comments,
      value: parseValue(value.join('=')),
    };
    comments = [];
  }

  if (!isValidServerConfig(config)) {
    throw new Error('Invalid server config');
  }

  return config;
}

export async function saveServerConfig(path: string, config: ServerConfig) {
  const lines: string[] = [];

  for (const [key, { comments, value }] of Object.entries(config)) {
    lines.push(...comments.map((comment) => comment.replace(new RegExp(`${EOL}$`), '')));
    lines.push(`${key}=${value}`);
    lines.push(EOL);
  }

  await fs.writeFile(path, lines.join(EOL));
  console.log(`${chalk.green('[SUCCESS]')} Saved server config to ${path}`);
}

export function getFromServerConfig(config: ServerConfig, key: 'Mods' | 'WorkshopItems'): string[] {
  return (config[key].value as string).split(';').filter((id) => id.length);
}
