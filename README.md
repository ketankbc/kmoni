# KMoni Phase 1

Minimal self-hosted Android device status monitor. The Go server serves the dashboard and accepts Android-initiated WebSockets. State is in memory; restarting the server disconnects devices but does not lose configuration.

## Run the server

Use HTTPS/WSS in production (put the container behind a TLS reverse proxy such as Caddy). Set `KMONI_PAIRING_TOKEN` in `config.env` to a random secret of at least 16 characters, then:

```sh
docker compose up -d --build
```

Open `https://your-host/`, sign in with the same token, and enter the HTTPS URL plus token in the Android app. Android accepts `https://` and automatically uses `wss://` for the WebSocket. Do not expose plain HTTP to the internet. Keep `config.env` private.

## Android build

Open `android/` in Android Studio and run the `app` configuration on a rooted or non-rooted Android device. The app only reports status in Phase 1; no root capability is used. Set the server URL and pairing token, then tap Connect. It reconnects with exponential backoff.

## Checks

```sh
go test ./...
docker compose build
```
