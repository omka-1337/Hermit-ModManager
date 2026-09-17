# BepInEx Mod Manager

Desktop mod manager for games using [BepInEx](https://github.com/BepInEx/BepInEx).
Built with [Wails v3](https://v3.wails.io) (Go + React/TypeScript). Linux only for now; Windows support is planned.

## Requirements (Arch Linux)

```sh
sudo pacman -S go nodejs npm gtk4 webkitgtk-6.0
go install github.com/wailsapp/wails/v3/cmd/wails3@latest
```

Make sure `~/go/bin` is in your `PATH`. Run `wails3 doctor` to verify the setup.

## Development

```sh
wails3 dev      # run with hot reload
wails3 build    # production build -> bin/
wails3 package  # AppImage / deb / rpm / arch packages
```

After changing Go services, bindings in `frontend/bindings` are regenerated automatically by `wails3 dev`/`wails3 build`
(or manually via `wails3 generate bindings -ts`).

## Layout

```
main.go              application entry point, window setup
internal/app/        app metadata and services exposed to the frontend
frontend/            React + TypeScript + Tailwind (Vite)
frontend/bindings/   generated TS bindings for Go services
build/               Wails build/packaging config (linux, windows, darwin)
```
