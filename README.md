# zrp-wallet-service

A minimal Go HTTP service that wraps the [zrp-wallet-sdk](https://github.com/LeonidasSpart/zrp-wallet-sdk) Solana package. It builds unsigned transactions and verifies signatures — **it never generates, receives, or stores private keys.**

## Architecture: fully non-custodial

Two supported client flows, neither of which touches this service for key material:

1. **Connect an existing wallet** (Phantom, Solflare, etc.) — the frontend uses [`@solana/wallet-adapter`](https://github.com/anza-xyz/wallet-adapter) to connect and sign. Keys never leave the user's wallet extension/app.
2. **In-app generated wallet** — the frontend generates a keypair client-side using [`@solana/web3.js`](https://github.com/solana-labs/solana-web3.js)'s `Keypair.generate()`, and the user is responsible for exporting/backing up the resulting seed phrase. This service is not involved in generation.

In both cases, this backend's job is narrow:

- **Build unsigned transactions** — construct the tx (instructions, fee payer, blockhash) so the frontend doesn't need to duplicate that logic, then hand it back for the client to sign with whichever wallet method they used
- **Verify signatures** — confirm a client-signed message or transaction is valid (e.g. for "Sign-In with Solana"-style auth, or sanity-checking a signed tx before you relay it)
- **Compute transaction hashes** — from a client-returned signed transaction, for logging/tracking before broadcast

This service also does **not** talk to the Solana network directly. Fetching balances, recent blockhashes, and broadcasting signed transactions happens via a public or private RPC endpoint, called from the frontend or your NestJS backend — not from here.

## Endpoints

### `GET /health`
Returns `{"status": "ok"}`.

### `POST /wallet/transfer/build`
Builds an unsigned native SOL transfer. Requires a recent blockhash fetched separately (e.g. via `getLatestBlockhash` on an RPC node).

```json
// request
{
  "from": "7NRmECq1R4tCtXNvmvDAuXmii3vN1J9DRZWhMCuuUnkM",
  "to": "2uWejjxZtzuqLrQeCH4gwh3C5TNn2rhHTdvC26dWzKfM",
  "amountLamports": 1000000000,
  "recentBlockhash": "Cfudd6AiXTzPYrmEBGNFsHgaNKJ3xrrsGCT39avLkoiu"
}

// response
{ "unsignedTransaction": "base58-encoded-unsigned-tx..." }
```

Hand `unsignedTransaction` to the frontend to sign — via `wallet.signTransaction()` (wallet adapter) or a locally-held keypair.

### `POST /wallet/transaction/hash`
Given a client-signed transaction, compute its hash.

```json
// request
{ "signedTransaction": "base58-encoded-signed-tx..." }

// response
{ "txHash": "..." }
```

### `POST /wallet/message/verify`
Verifies a message signature produced client-side.

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
4. Railway assigns a `PORT` env var automatically; the service reads it at startup

## Local development

Requires Go 1.23+ and network access to fetch the SDK dependency:

```shell
go get github.com/LeonidasSpart/zrp-wallet-sdk/coins/solana@main
go mod tidy
go run .
```
