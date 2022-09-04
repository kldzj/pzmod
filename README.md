## pzmod

![Project Banner showing usage example](/banner.png?raw=true)

**pzmod** is a simple Project Zomboid server mods manager that allows you to easily install and manage mods on your server.

### Features

- Validates mods against the Steam Workshop API
- Hints at problems with in your mod list like missing dependencies
- Allows you to safely install mods from the Steam Workshop
- Allows you to safely remove mods

### Usage

```bash
pzmod --file path/to/servertest.ini
```

### Requirements

- Steam API Key ([see here](https://steamcommunity.com/dev/apikey))
- Installed Project Zomboid server (or at least a `servertest.ini` file)

### Download

You can download the latest version of **pzmod** from the [releases page](https://github.com/kldzj/pzmod/releases).

### Hint

Dependencies are not automatically installed. After adding a mod that has dependencies, you'll see an error message including the Workshop IDs of the missing dependencies. You have to install them manually.
