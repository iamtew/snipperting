# snipperting

![snipperting](docs/logo-sm.png)

Windows tray snippets. Click a note to copy it, or inject it at the caret.

[Website](https://iamtew.github.io/snipperting/) has the product page and manual. Help in the app shows the same guide.

## Repo

Windows, Go 1.26, [Just](https://github.com/casey/just). Editor needs Monaco once:

```
just vendor-monaco
just test
just run
```

`just build` writes `dist/snipperting.exe`. Other recipes: `just --list`.

MIT — see [LICENSE](LICENSE).
