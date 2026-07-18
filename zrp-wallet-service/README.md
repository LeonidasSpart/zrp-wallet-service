# zrp-wallet-service

A minimal Go HTTP service that wraps the [zrp-wallet-sdk](https://github.com/LeonidasSpart/zrp-wallet-sdk) Solana package for address generation, offline transaction signing, and message signing/verification.

**This service does not talk to the Solana network.** It has no RPC client — fetching balances, recent blockhashes, or broadcasting signed transactions is the caller's responsibility (e.g. your NestJS backend, or directly from a frontend using a public/private RPC endpoint).

## ⚠️ Security notes (read before deploying to production)

- Private keys are accepted as plain request bodies and returned in plain responses. This service **must** run behind HTTPS only (Railway provides this by default).
- This service does **not** persist private keys anywhere — it's stateless. Whoever calls it is responsible for secure storage (e.g. encrypted at rest, hardware wallet, or client-side only).
- Do not log request/response bodies in production — they contain private keys. Consider adding request logging middleware that redacts `privateKey`/`fromPrivateKeyBase58` fields if you add logging later.
- This is an MVP for internal/testing use. Before handling real funds, consider: rate limiting, auth on these endpoints (they should never be publicly reachable without an API key or internal network restriction), and an audit of the signing flow.

## Endpoints

### `GET /health`
Returns `{"status": "ok"}`.

### `POST /wallet/generate`
Generates a new Solana keypair.

```json
// response
{
  "address": "7NRmECq1R4tCtXNvmvDAuXmii3vN1J9DRZWhMCuuUnkM",
  "privateKey": "a1b2c3..." // hex-encoded, store securely
}
```

### `POST /wallet/transfer/sign`
Builds and signs a native SOL transfer offline. You must supply a recent blockhash fetched separately from an RPC node (e.g. via `getLatestBlockhash`).

```json
// request
{
  "fromPrivateKeyBase58": "tzyJiBd5PzFPFfVnnfVx14rsfC8FKW8idpJwNhH6FxzZAdhgBp4CrDxcUW9D89f5k3W6WhVnybbAw7RRB2HPxnt",
  "to": "7NRmECq1R4tCtXNvmvDAuXmii3vN1J9DRZWhMCuuUnkM",
  "amountLamports": 1000000000,
  "recentBlockhash": "Cfudd6AiXTzPYrmEBGNFsHgaNKJ3xrrsGCT39avLkoiu"
}

// response
{
  "signedTransaction": "base58-encoded-signed-tx...",
  "txHash": "..."
}
```

Broadcast `signedTransaction` yourself via an RPC node's `sendTransaction` method.

### `POST /wallet/message/sign`
```json
// request
{
  "privateKeyBase58": "tzyJiBd5...",
  "message": "this is a message to be signed by solana",
  "utf8": true
}

// response
{ "signature": "..." }
```

### `POST /wallet/message/verify`
```json
// request
{
  "address": "2uWejjxZtzuqLrQeCH4gwh3C5TNn2rhHTdvC26dWzKfM",
  "message": "this is a message to be signed by solana",
  "signature": "...",
  "utf8": true
}

// response
{ "valid": true }
```

## Setup

1. Push this repo to GitHub
2. Go to **Actions → Build and Resolve Dependencies → Run workflow** — this fetches the `zrp-wallet-sdk` dependency, runs `go mod tidy`, verifies the build, and commits the resulting `go.mod`/`go.sum` back to the repo
3. Once that workflow succeeds, connect the repo to Railway — `railway.toml` is already configured for the build/start commands and a `/health` healthcheck
4. Railway will assign a `PORT` env var automatically; the service reads it at startup

## Local development

Requires Go 1.23+ and network access to fetch the SDK dependency:

```shell
go get github.com/LeonidasSpart/zrp-wallet-sdk/coins/solana@main
go mod tidy
go run .
```
