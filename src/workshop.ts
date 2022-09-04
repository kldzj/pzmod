import fetch from '@adobe/node-fetch-retry';

export interface ModEntry {
  workshopId: string;
  modIds: string[];
  children: string[];
  creator: string;
  title: string;
  description: string;
  tags: string[];
  banned: boolean;
}

const cache = new Map<string, ModEntry>();

function chunk<T>(array: T[], size: number): T[][] {
  const chunks: T[][] = [];
  for (let i = 0; i < array.length; i += size) {
    chunks.push(array.slice(i, i + size));
  }
  return chunks;
}

function parseModIds(description: string): string[] {
  const regex = /Mod ID: (.+)\n?$/gm;
  const matches = [...description.matchAll(regex)];
  return matches.map((match) => match[1]);
}

function getURLParams(key: string, workshopIds: string[]): string {
  return new URLSearchParams({
    key,
    includetags: 'true',
    includechildren: 'true',
    ...workshopIds.reduce((acc, id, i) => ({ ...acc, [`publishedfileids[${i}]`]: id }), {}),
  }).toString();
}

async function getPublishedItemsDetailsChunk(key: string, workshopIds: string[]): Promise<ModEntry[]> {
  const idsToFetch = workshopIds.filter((id) => !cache.has(id));
  if (!idsToFetch.length) {
    return workshopIds.map((id) => cache.get(id)!);
  }

  const unfetchedIds = workshopIds.filter((id) => !idsToFetch.includes(id));
  const url = `https://api.steampowered.com/IPublishedFileService/GetDetails/v1/?${getURLParams(key, idsToFetch)}`;
  const response = await fetch(url, {
    headers: {
      Accept: 'application/json',
    },
  });

  if (!response.ok) {
    throw new Error(`Failed to fetch Workshop Items: ${response.statusText}`);
  }

  const json = await response.json();
  const items = json.response.publishedfiledetails;
  if (!items) {
    throw new Error(`Failed to fetch Workshop Items: ${json.response?.error ?? 'Unknown error'}`);
  }

  const fetched = items.map((item: any) => ({
    workshopId: item.publishedfileid,
    modIds: parseModIds(item.file_description),
    children: item.children?.map((child: any) => child.publishedfileid) ?? [],
    creator: item.creator,
    title: item.title,
    description: item.file_description,
    tags: item.tags?.map((tag: any) => tag.display_name) ?? [],
    banned: item.banned,
  }));

  for (const item of fetched) {
    cache.set(item.workshopId, item);
  }

  return [...unfetchedIds.map((id) => cache.get(id)), ...fetched];
}

export async function getPublishedItemsDetails(key: string, workshopIds: string[]): Promise<ModEntry[]> {
  if (!workshopIds.length) return [];
  const workshopIdsChunks = chunk(workshopIds, 10);
  const workshopItems = await Promise.all(workshopIdsChunks.map((ids) => getPublishedItemsDetailsChunk(key, ids)));
  return workshopItems.flat();
}
