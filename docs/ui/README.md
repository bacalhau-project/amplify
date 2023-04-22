# UI

## Running In Dev Mode

In one shell:

```bash
cd ui
yarn build
yarn dev
```

In another shell:

```bash
# disable-cors is only required when running the UI separately
go run . serve --disable-cors
```

Browse to http://localhost:5173/

Edit the UI and it will autoreload.
