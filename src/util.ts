import chalk from 'chalk';
import { ModEntry } from './workshop';

export function chunk<T>(array: T[], size: number): T[][] {
  const chunks: T[][] = [];
  for (let i = 0; i < array.length; i += size) {
    chunks.push(array.slice(i, i + size));
  }

  return chunks;
}

export function dedupe<T>(array: T[]): T[] {
  return [...new Set(array)];
}

export function workshopURL(workshopId: string): string {
  return `https://steamcommunity.com/sharedfiles/filedetails/?id=${workshopId}`;
}

export function checkForUnusedDependencies(items: ModEntry[], children: ModEntry[]) {
  const missing = children.filter((child) => !items.find((item) => item.workshopId === child.workshopId));
  for (const item of missing) {
    const dependents = items.filter((_item) => _item.children.includes(item.workshopId));
    console.log(
      `${chalk.red('[ERROR]')} Missing dependency: ${chalk.underline(item.title)} (${
        item.workshopId
      }), required by: ${dependents.map((item) => chalk.underline(item.title)).join(' and ')}`
    );
  }
}

export function checkForUnusedMods(modIds: string[], items: ModEntry[], children: ModEntry[]) {
  for (const item of [...items, ...children]) {
    for (const modId of item.modIds) {
      if (!modIds.includes(modId)) {
        console.warn(
          `${chalk.yellow('[WARN]')} Ununsed mod ID: '${chalk.underline(modId)}' - ${item.title} (${item.workshopId})`
        );
      }
    }
  }
}

export function checkForUnknownMods(modIds: string[], items: ModEntry[]) {
  for (const modId of modIds) {
    if (!items.find((item) => item.modIds.includes(modId))) {
      console.warn(
        `${chalk.yellow('[WARN]')} Unknown mod ID: '${chalk.underline(modId)}' (not found in WorkshopItems)`
      );
    }
  }
}

export function filterExistingModIds(modIds: string[], item: ModEntry) {
  return modIds.filter((modId) => !item.modIds.includes(modId));
}
