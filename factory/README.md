# Потік

Спокійна гра про фабрику з конвеєрами. Працює офлайн і без встановлення: один `potik.exe`.
Задум і план описані в [DESIGN.md](DESIGN.md).

## Збірка

Потрібен Go 1.26+.

```sh
# Windows (з будь-якої ОС)
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w -H windowsgui" -o dist/potik.exe ./cmd/potik

# тести симуляції
go test ./...
```

Збереження лежить у `%AppData%\Potik\solo.sav`. Щоб почати новий світ, запусти `potik.exe -new`: старий сейв перейменується в `.bak`.

## Структура

- `internal/sim`: симуляція (світ, поклади, конвеєри, машини, збереження), без графіки.
- `internal/game`: рушій Ebitengine: камера, малювання, керування, інтерфейс.
- `cmd/potik`: точка входу.
