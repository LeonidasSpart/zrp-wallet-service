package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/LeonidasSpart/zrp-wallet-sdk/coins/solana"
	"github.com/LeonidasSpart/zrp-wallet-sdk/coins/solana/base"
)

// ---------- helpers ----------

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

// ---------- POST /wallet/transfer/build ----------
// Builds an UNSIGNED native SOL transfer. No private key is ever involved —
// the caller signs the returned transaction client-side (via a connected
// wallet like Phantom, or a client-held keypair) and either broadcasts it
// directly or sends the signed tx to /wallet/transaction/hash.
//
// The caller must supply a recent blockhash (fetched from an RPC node —
// e.g. api.mainnet-beta.solana.com's getLatestBlockhash) since this service
// never talks to the network.

type buildTransferRequest struct {
	From            string `json:"from"`
	To              string `json:"to"`
	AmountLamports  uint64 `json:"amountLamports"`
	RecentBlockhash string `json:"recentBlockhash"`
}

type buildTransferResponse struct {
	UnsignedTransaction string `json:"unsignedTransaction"` // base58, hand to client to sign
}

func buildTransferHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, errMethodNotAllowed)
		return
	}

	var req buildTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if !base.ValidateAddress(req.From) || !base.ValidateAddress(req.To) {
		writeError(w, http.StatusBadRequest, errInvalidAddress)
		return
	}

	rawTx := solana.NewRawTransaction(req.RecentBlockhash, req.From)
	rawTx.AppendTransferInstruction(req.AmountLamports, req.From, req.To)

	unsignedTx, err := rawTx.UnsignedTx()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, buildTransferResponse{
		UnsignedTransaction: unsignedTx,
	})
}

// ---------- POST /wallet/transaction/hash ----------
// Given a client-signed transaction (e.g. returned from a wallet adapter
// after the user approves it), compute its hash. Useful for tracking before
// you broadcast it, or logging on your backend without needing a private key.

type txHashRequest struct {
	SignedTransaction string `json:"signedTransaction"` // base58
}

type txHashResponse struct {
	TxHash string `json:"txHash"`
}

func txHashHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, errMethodNotAllowed)
		return
	}

	var req txHashRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	txHash, err := solana.CalTxHash(req.SignedTransaction)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusOK, txHashResponse{TxHash: txHash})
}

// ---------- POST /wallet/message/verify ----------
// Verifies a message signature produced client-side (e.g. by a connected
// wallet's signMessage() call, used for "Sign-In with Solana"-style auth).
// No private key is ever involved.

type verifyMessageRequest struct {
	Address   string `json:"address"`
	Message   string `json:"message"`
	Signature string `json:"signature"`
	Utf8      bool   `json:"utf8"`
}

func verifyMessageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, errMethodNotAllowed)
		return
	}

	var req verifyMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var err error
	if req.Utf8 {
		err = solana.VerifySignedUtf8Message(req.Address, req.Message, req.Signature)
	} else {
		err = solana.VerifySignedMessage(req.Address, req.Message, req.Signature)
	}

	writeJSON(w, http.StatusOK, map[string]bool{"valid": err == nil})
}

// ---------- GET /health ----------

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ---------- wiring ----------

var errMethodNotAllowed = &simpleError{"method not allowed"}
var errInvalidAddress = &simpleError{"invalid solana address"}

type simpleError struct{ msg string }

func (e *simpleError) Error() string { return e.msg }

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/wallet/transfer/build", buildTransferHandler)
	mux.HandleFunc("/wallet/transaction/hash", txHashHandler)
	mux.HandleFunc("/wallet/message/verify", verifyMessageHandler)

	log.Printf("zrp-wallet-service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
