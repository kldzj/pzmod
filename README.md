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
# Will launch in interactive mode
pzmod --file path/to/servertest.ini

# Will set the servers public name
pzmod --file path/to/servertest.ini set name "My Server"

# Will list the available keys to set through the CLI
pzmod --file path/to/servertest.ini get list
```

A list of available commands can be found by running `pzmod --help`.

### Requirements

- Steam API Key ([see here](https://steamcommunity.com/dev/apikey))
- Installed Project Zomboid server (or at least a `servertest.ini` file)

### Download

You can download the latest version of **pzmod** from the [releases page](https://github.com/kldzj/pzmod/releases).

> Note: v2 is currently in pre-release. Bug reports and feedback are welcome.
