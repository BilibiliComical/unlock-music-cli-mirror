# Unlock Music Project - CLI Edition

Original: Web Edition https://git.unlock-music.dev/um/web

- [![Build Status](https://ci.unlock-music.dev/api/badges/um/cli/status.svg)](https://ci.unlock-music.dev/um/cli)
- [Release Download](https://github.com/BilibiliComical/unlock-music-cli-mirror/releases)
- [Latest Build](https://git.unlock-music.dev/um/-/packages/generic/cli-build/)

> **WARNING**
> 

## Features

- [x] All Algorithm Supported By `unlock-music/web`
- [x] Complete Metadata & Cover Image

## Hou to Build 如何构建

- Requirements（要求）: **Golang 1.19**

1. run `go install unlock-music.dev/cli/cmd/um@master`

## How to use 如何使用

- Drag the encrypted file to `um.exe` (Tested on Windows)
- Run: `./um [-o <output dir>] [-i] <input dir/file>`
- Use `./um -h` to show help menu

## Update CI pipeline

1. Install [Drone CI binary](https://docs.drone.io/cli/install/)
2. Edit `.drone.jsonnet`
3. Update drone CI pipeline:

   ```sh
   drone jsonnet --format --stream
   ```
