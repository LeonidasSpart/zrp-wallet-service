package main

import (
	"encoding/hex"
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

// ---------- POST /wallet/generate ----------
// Generates a brand-new Solana keypair. The caller is responsible for
// storing the private key securely — this service does not persist it.

type generateResponse struct {
	Address    string `json:"address"`
	PrivateKey string `json:"privateKey"` // hex-encoded
}

func generateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, errMethodNotAllowed)
		return
	}

	pk, err := base.NewRandomPrivateKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	privHex := hex.EncodeToString(pk.Bytes())
	address, err := solana.NewAddress(privHex)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, generateResponse{
		Address:    address,
		PrivateKey: privHex,
	})
}

// ---------- POST /wallet/transfer/sign ----------
// Builds and signs a native SOL transfer. The caller must supply a recent
// blockhash (fetched from an RPC node — e.g. api.mainnet-beta.solana.com's
// getLatestBlockhash) since this service never talks to the network.

type transferRequest struct {
	FromPrivateKeyBase58 string `json:"fromPrivateKeyBase58"`
	To                   string `json:"to"`
	AmountLamports       uint64 `json:"amountLamports"`
	RecentBlockhash      string `json:"recentBlockhash"`
}

type transferResponse struct {
	SignedTransaction string `json:"signedTransaction"` // base58
	TxHash            string `json:"txHash"`
}

func transferSignHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, errMethodNotAllowed)
		return
	}

	var req transferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	fromPrivate, err := base.PrivateKeyFromBase58(req.FromPrivateKeyBase58)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	from := fromPrivate.PublicKey().String()

	rawTx := solana.NewRawTransaction(req.RecentBlockhash, from)
	rawTx.AppendTransferInstruction(req.AmountLamports, from, req.To)
	rawTx.AppendSigner(hex.EncodeToString(fromPrivate.Bytes()))

	signedTx, err := rawTx.Sign(true) // true = return base58-encoded
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	txHash, err := solana.CalTxHash(signedTx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, transferResponse{
		SignedTransaction: signedTx,
		TxHash:            txHash,
	})
}

// ---------- POST /wallet/message/sign ----------

type signMessageRequest struct {
	PrivateKeyBase58 string `json:"privateKeyBase58"`
	Message          string `json:"message"`
	Utf8             bool   `json:"utf8"` // true = plain utf-8 text, false = base58-encoded message
}

type signMessageResponse struct {
	Signature string `json:"signature"`
}

func signMessageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, errMethodNotAllowed)
		return
	}

	var req signMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var (
		sig string
		err error
	)
	if req.Utf8 {
		sig, err = solana.SignUtf8Message(req.PrivateKeyBase58, req.Message)
	} else {
		sig, err = solana.SignMessage(req.PrivateKeyBase58, req.Message)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, signMessageResponse{Signature: sig})
}

// ---------- POST /wallet/message/verify ----------

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

var errMethodNotAllowed = &methodNotAllowedError{}

type methodNotAllowedError struct{}

func (e *methodNotAllowedError) Error() string { return "method not allowed" }

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/wallet/generate", generateHandler)
	mux.HandleFunc("/wallet/transfer/sign", transferSignHandler)
	mux.HandleFunc("/wallet/message/sign", signMessageHandler)
	mux.HandleFunc("/wallet/message/verify", verifyMessageHandler)

	log.Printf("zrp-wallet-service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
