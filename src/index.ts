import { EOL } from 'os';
import { resolve } from 'path';
import { existsSync } from 'fs';
import terminalLink from 'terminal-link';
import inquirer from 'inquirer';
import chalk from 'chalk';
import arg from 'arg';
import { Config, loadConfig, saveConfig } from './config';
import { getFromServerConfig, readServerConfig, saveServerConfig } from './ini';
import { getPublishedItemsDetails } from './workshop';
import {
  checkForUnknownMods,
  checkForUnusedDependencies,
  checkForUnusedMods,
  filterExistingModIds,
  workshopURL,
} from './util';

inquirer.registerPrompt('checkbox-plus', require('inquirer-checkbox-plus-prompt'));

const args = arg({ '--file': String });

const START_OF_LIST = '__pzmod__start__of__list__';
const END_OF_LIST = '__pzmod__end__of__list__';

async function main() {
  const file = args['--file'];
  if (!file) {
    console.log(
      'Please use --file to specify the path to a Project Zomboid server config (e.g. servertest.ini) to read'
    );
    return;
  }

  const path = resolve(file);
  if (!existsSync(path)) {
    console.log(`File not found: ${path}`);
    return;
  }

  let ended = false;
  while (!ended) {
    let config: Config;
    while (!(config = await loadConfig()).apiKey) {
      const { apiKey } = await inquirer.prompt([
        {
          type: 'password',
          name: 'apiKey',
          message: 'Please enter your Steam API key:',
        },
      ]);

      if (!apiKey) {
        console.log('API key cannot be empty');
        continue;
      }

      await saveConfig({ ...config, apiKey });
    }
    // -- end api key prompt --

    const serverConfig = await readServerConfig(path, true);
    const { action } = await inquirer.prompt([
      {
        type: 'list',
        name: 'action',
        message: 'What would you like to do?',
        choices: [
          {
            name: 'Show or modify mod list',
            value: 'mods',
            checked: true,
          },
          {
            name: 'Delete Steam API key',
            value: 'delete_key',
          },
          {
            name: 'Exit',
            value: 'exit',
          },
        ],
      },
    ]);

    switch (action) {
      case 'mods': {
        let backToMain = false;
        while (!backToMain) {
          console.log(chalk.gray('Loading Workshop Item details...'));
          const items = await getPublishedItemsDetails(
            config.apiKey!,
            getFromServerConfig(serverConfig, 'WorkshopItems')
          );
          const childIds = items.reduce<string[]>(
            (acc, item) => (item.children.length ? [...acc, ...item.children] : acc),
            []
          );

          const children = await getPublishedItemsDetails(config.apiKey!, childIds);
          checkForUnknownMods(getFromServerConfig(serverConfig, 'Mods'), items);
          checkForUnusedMods(getFromServerConfig(serverConfig, 'Mods'), items, children);
          checkForUnusedDependencies(items, children);

          const { action } = await inquirer.prompt([
            {
              type: 'list',
              name: 'action',
              message: 'What would you like to do?',
              choices: [
                {
                  name: 'List Mods',
                  value: 'list',
                },
                {
                  name: 'Add Mod(s)',
                  value: 'add',
                },
                {
                  name: 'Remove Mods',
                  value: 'remove',
                },
                {
                  name: 'Write config file (save changes)',
                  value: 'save',
                },
              ],
            },
          ]);

          switch (action) {
            case 'list': {
              console.log(chalk.bold(`Listing ${items.length} mods:`));
              const modIds = getFromServerConfig(serverConfig, 'Mods');
              const workshopIds = getFromServerConfig(serverConfig, 'WorkshopItems');
              for (const item of items) {
                console.log(`${chalk.bold.underline(item.title)} (${item.workshopId})`);
                console.log(terminalLink('Open Workshop Page', workshopURL(item.workshopId)));
                if (item.banned) {
                  console.warn(`${chalk.yellow('[WARN]')} This mod has been ${chalk.red('banned')} from the workshop!`);
                }

                console.log(`Dependencies: ${!item.children.length ? chalk.gray('None') : ''}`);
                if (item.children.length) {
                  for (const child of item.children) {
                    const childItem = children.find((item) => item.workshopId === child);
                    if (childItem) {
                      const installed = workshopIds.includes(childItem.workshopId);
                      const enabled = installed && childItem.modIds.some((modId) => modIds.includes(modId));
                      console.log(
                        `  - ${
                          !installed
                            ? chalk.red(childItem.title)
                            : !enabled
                            ? chalk.yellow(childItem.title)
                            : childItem.title
                        } (${chalk.gray(childItem.workshopId)})`
                      );
                    } else {
                      console.log(`  - ${chalk.red(child)} (missing from workshop)`);
                    }
                  }
                }

                console.log(EOL);
              }

              console.log(`Dependencies marked ${chalk.yellow('yellow')} are installed but not enabled.`);
              console.log(`Dependencies marked ${chalk.red('red')} are not installed.`);

              const { more } = await inquirer.prompt([
                {
                  type: 'confirm',
                  name: 'more',
                  message: 'Would you like to do more?',
                },
              ]);

              if (!more) {
                backToMain = true;
              }

              break;
            }
            case 'add': {
              let more = true;
              while (more) {
                const modIds = getFromServerConfig(serverConfig, 'Mods');
                const workshopIds = getFromServerConfig(serverConfig, 'WorkshopItems');

                const { workshopId } = await inquirer.prompt([
                  {
                    type: 'input',
                    name: 'workshopId',
                    message: 'Please enter the Workshop ID of the mod you would like to add:',
                  },
                ]);

                if (!workshopId) {
                  console.error(`${chalk.red('[ERROR]')} Workshop ID cannot be empty`);
                  continue;
                }

                const [item] = await getPublishedItemsDetails(config.apiKey!, [workshopId]);
                if (!item) {
                  console.error(`${chalk.red('[ERROR]')} Workshop Item not found`);
                  continue;
                }

                const { newModIds, addAfter } = await inquirer.prompt([
                  {
                    type: 'checkbox',
                    name: 'newModIds',
                    message: 'Please select the mod(s) you would like to enable:',
                    choices: item.modIds.map((modId) => ({
                      name: modId,
                      value: modId,
                      checked: modIds.includes(modId) || item.modIds.length === 1,
                    })),
                  },
                  {
                    type: 'list',
                    name: 'addAfter',
                    message: 'Where would you like to add this mod?',
                    choices: [
                      {
                        name: 'Add to the beginning of the list',
                        value: START_OF_LIST,
                      },
                      {
                        name: 'Add to the end of the list',
                        value: END_OF_LIST,
                      },
                      ...modIds
                        .filter((modId) => !item.modIds.includes(modId))
                        .map((modId) => ({ name: `After ${chalk.bold(modId)}`, value: modId })),
                    ],
                  },
                ]);

                if (!newModIds.length) {
                  console.error(`${chalk.yellow('[WARN]')} No mods selected`);
                  continue;
                }

                if (!workshopIds.includes(item.workshopId)) {
                  serverConfig.WorkshopItems.value = [...workshopIds, item.workshopId].join(';');
                }

                const withoutExisting = filterExistingModIds(modIds, item);
                if (addAfter === END_OF_LIST) {
                  serverConfig.Mods.value = [...withoutExisting, ...newModIds].join(';');
                } else if (addAfter === START_OF_LIST) {
                  serverConfig.Mods.value = [...newModIds, ...withoutExisting].join(';');
                } else {
                  const index = withoutExisting.indexOf(addAfter);
                  withoutExisting.splice(index + 1, 0, ...newModIds);
                  serverConfig.Mods.value = withoutExisting.join(';');
                }

                console.log(`${chalk.green('[SUCCESS]')} Mod ${chalk.bold(item.title)} (${item.workshopId}) added`);
                if (item.children.length) {
                  for (const child of item.children) {
                    const childInstalled = items.find((item) => item.workshopId === child);
                    if (!childInstalled) {
                      const [childItem] = await getPublishedItemsDetails(config.apiKey!, [child]);
                      console.log(
                        `${chalk.yellow('[WARN]')} Newly added mod is missing dependency ${chalk.underline(
                          childItem.title
                        )} (${childItem.workshopId})`
                      );
                      continue;
                    }
                  }
                }

                const { addMore } = await inquirer.prompt([
                  {
                    type: 'confirm',
                    name: 'addMore',
                    message: 'Would you like to add more mods?',
                  },
                ]);

                more = addMore;
              }

              break;
            }
            case 'remove': {
              const { workshopIds } = await inquirer.prompt([
                {
                  type: 'checkbox-plus',
                  name: 'workshopIds',
                  message: 'Please select the mod(s) you would like to remove:',
                  searchable: true,
                  highlight: true,
                  source: async (_answersSoFar: any, input = '') => {
                    return items
                      .filter(
                        (item) =>
                          item.title.toLowerCase().includes(input.toLowerCase()) || item.workshopId.includes(input)
                      )
                      .map((item) => ({
                        name: `${item.title} (${item.workshopId})`,
                        value: item.workshopId,
                      }));
                  },
                },
              ]);

              if (!workshopIds.length) {
                console.error(`${chalk.yellow('[WARN]')} No mods selected`);
                break;
              }

              const modIds = getFromServerConfig(serverConfig, 'Mods');
              const modIdsToFilter = items
                .filter((item) => workshopIds.includes(item.workshopId))
                .reduce<string[]>(
                  (acc, item) => [...acc, ...item.modIds.filter((modId) => modIds.includes(modId))],
                  []
                );

              serverConfig.Mods.value = getFromServerConfig(serverConfig, 'Mods')
                .filter((modId) => !modIdsToFilter.includes(modId))
                .join(';');

              serverConfig.WorkshopItems.value = getFromServerConfig(serverConfig, 'WorkshopItems')
                .filter((workshopId) => !workshopIds.includes(workshopId))
                .join(';');

              console.log(
                `${chalk.green('[SUCCESS]')} ${modIdsToFilter.length} mod(s) (${
                  workshopIds.length
                } Workshop Item(s)) removed`
              );

              break;
            }
            case 'save': {
              await saveServerConfig(path, serverConfig);
              backToMain = true;
              break;
            }
            default: {
              console.warn(`Unknown action '${action}'`);
              break;
            }
          }

          console.log(EOL);
        }

        break;
      }
      case 'delete_key': {
        const { confirm } = await inquirer.prompt([
          {
            type: 'confirm',
            name: 'confirm',
            message: 'Are you sure you want to delete your Steam API key?',
          },
        ]);

        if (confirm) {
          const { apiKey, ...newConfig } = config;
          await saveConfig(newConfig);
          console.log(`${chalk.yellow('[WARN]')} API key deleted`);
        } else {
          console.log(`${chalk.yellow('[WARN]')} API key ${chalk.underline('not')} deleted`);
        }

        break;
      }
      case 'exit':
        ended = true;
        break;
      default: {
        console.warn(`Unknown action '${action}'`);
        break;
      }
    }
  }
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
