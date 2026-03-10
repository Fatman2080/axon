package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type FaucetServer struct {
	client    *ethclient.Client
	key       *ecdsa.PrivateKey
	address   common.Address
	chainID   *big.Int
	dripWei   *big.Int
	cooldown  time.Duration
	mu        sync.Mutex
	lastDrip  map[string]time.Time
	txCount   uint64
	startTime time.Time
}

type DripRequest struct {
	Address string `json:"address"`
}

type DripResponse struct {
	Success bool   `json:"success"`
	TxHash  string `json:"tx_hash,omitempty"`
	Amount  string `json:"amount,omitempty"`
	Message string `json:"message,omitempty"`
}

type StatusResponse struct {
	Address     string `json:"faucet_address"`
	Balance     string `json:"balance"`
	DripAmount  string `json:"drip_amount"`
	Cooldown    string `json:"cooldown"`
	ChainID     string `json:"chain_id"`
	TotalDrips  uint64 `json:"total_drips"`
	UptimeHours string `json:"uptime_hours"`
}

func main() {
	rpcURL := envOrDefault("RPC_URL", "http://localhost:8545")
	chainIDStr := envOrDefault("CHAIN_ID", "9001")
	dripAmount := envOrDefault("DRIP_AMOUNT", "10")
	cooldownSec := envOrDefault("COOLDOWN_SECONDS", "86400")
	port := envOrDefault("PORT", "8080")

	privateKeyHex := os.Getenv("FAUCET_PRIVATE_KEY")
	if privateKeyHex == "" {
		keyFile := os.Getenv("FAUCET_PRIVATE_KEY_FILE")
		if keyFile != "" {
			data, err := os.ReadFile(keyFile)
			if err != nil {
				log.Fatalf("Failed to read key file: %v", err)
			}
			privateKeyHex = strings.TrimSpace(string(data))
		}
	}
	if privateKeyHex == "" {
		log.Fatal("FAUCET_PRIVATE_KEY or FAUCET_PRIVATE_KEY_FILE must be set")
	}
	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")

	key, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		log.Fatalf("Invalid private key: %v", err)
	}
	address := crypto.PubkeyToAddress(key.PublicKey)

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Fatalf("Failed to connect to RPC: %v", err)
	}

	chainID, _ := new(big.Int).SetString(chainIDStr, 10)

	// dripAmount in AXON -> convert to aaxon (1e18)
	dripAXON, _ := new(big.Int).SetString(dripAmount, 10)
	dripWei := new(big.Int).Mul(dripAXON, new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))

	cd, _ := strconv.Atoi(cooldownSec)

	faucet := &FaucetServer{
		client:    client,
		key:       key,
		address:   address,
		chainID:   chainID,
		dripWei:   dripWei,
		cooldown:  time.Duration(cd) * time.Second,
		lastDrip:  make(map[string]time.Time),
		startTime: time.Now(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", faucet.handleIndex)
	mux.HandleFunc("/api/faucet", faucet.handleDrip)
	mux.HandleFunc("/api/status", faucet.handleStatus)
	mux.HandleFunc("/health", faucet.handleHealth)

	handler := corsMiddleware(mux)

	log.Printf("Axon Faucet starting on :%s", port)
	log.Printf("  Address: %s", address.Hex())
	log.Printf("  Drip:    %s AXON", dripAmount)
	log.Printf("  RPC:     %s", rpcURL)

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}

func (f *FaucetServer) handleDrip(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, DripResponse{
			Message: "POST only",
		})
		return
	}

	var req DripRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, DripResponse{
			Message: "invalid JSON body, expected {\"address\": \"0x...\"}",
		})
		return
	}

	if !common.IsHexAddress(req.Address) {
		writeJSON(w, http.StatusBadRequest, DripResponse{
			Message: "invalid Ethereum address",
		})
		return
	}

	addr := common.HexToAddress(req.Address)
	addrLower := strings.ToLower(addr.Hex())

	f.mu.Lock()
	last, exists := f.lastDrip[addrLower]
	if exists && time.Since(last) < f.cooldown {
		remaining := f.cooldown - time.Since(last)
		f.mu.Unlock()
		writeJSON(w, http.StatusTooManyRequests, DripResponse{
			Message: fmt.Sprintf("cooldown active, try again in %s", remaining.Round(time.Second)),
		})
		return
	}
	f.lastDrip[addrLower] = time.Now()
	f.mu.Unlock()

	txHash, err := f.sendTx(addr)
	if err != nil {
		f.mu.Lock()
		delete(f.lastDrip, addrLower)
		f.mu.Unlock()

		log.Printf("ERROR drip to %s: %v", addr.Hex(), err)
		writeJSON(w, http.StatusInternalServerError, DripResponse{
			Message: fmt.Sprintf("transaction failed: %v", err),
		})
		return
	}

	f.mu.Lock()
	f.txCount++
	f.mu.Unlock()

	dripAXON := new(big.Int).Div(f.dripWei, new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))

	log.Printf("DRIP %s AXON -> %s (tx: %s)", dripAXON.String(), addr.Hex(), txHash)
	writeJSON(w, http.StatusOK, DripResponse{
		Success: true,
		TxHash:  txHash,
		Amount:  dripAXON.String() + " AXON",
		Message: "tokens sent successfully",
	})
}

func (f *FaucetServer) sendTx(to common.Address) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	nonce, err := f.client.PendingNonceAt(ctx, f.address)
	if err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}

	gasPrice, err := f.client.SuggestGasPrice(ctx)
	if err != nil {
		gasPrice = big.NewInt(0)
	}

	tx := types.NewTransaction(nonce, to, f.dripWei, 21000, gasPrice, nil)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(f.chainID), f.key)
	if err != nil {
		return "", fmt.Errorf("sign: %w", err)
	}

	if err := f.client.SendTransaction(ctx, signedTx); err != nil {
		return "", fmt.Errorf("send: %w", err)
	}

	return signedTx.Hash().Hex(), nil
}

func (f *FaucetServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bal, err := f.client.BalanceAt(ctx, f.address, nil)
	if err != nil {
		bal = big.NewInt(0)
	}

	balAXON := new(big.Int).Div(bal, new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	dripAXON := new(big.Int).Div(f.dripWei, new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	uptime := time.Since(f.startTime).Hours()

	f.mu.Lock()
	txCount := f.txCount
	f.mu.Unlock()

	writeJSON(w, http.StatusOK, StatusResponse{
		Address:     f.address.Hex(),
		Balance:     balAXON.String() + " AXON",
		DripAmount:  dripAXON.String() + " AXON",
		Cooldown:    f.cooldown.String(),
		ChainID:     f.chainID.String(),
		TotalDrips:  txCount,
		UptimeHours: fmt.Sprintf("%.1f", uptime),
	})
}

func (f *FaucetServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := f.client.BlockNumber(ctx)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"status":"unhealthy","error":"%s"}`, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status":"healthy"}`)
}

func (f *FaucetServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	dripAXON := new(big.Int).Div(f.dripWei, new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	fmt.Fprintf(w, faucetHTML, dripAXON.String(), f.cooldown.String())
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

const faucetHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Axon Testnet Faucet</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;
  background:linear-gradient(135deg,#0f0c29,#302b63,#24243e);color:#e0e0e0;
  min-height:100vh;display:flex;align-items:center;justify-content:center}
.card{background:rgba(255,255,255,0.06);backdrop-filter:blur(12px);
  border:1px solid rgba(255,255,255,0.1);border-radius:20px;
  padding:48px 40px;max-width:500px;width:90%%;text-align:center}
h1{font-size:28px;margin-bottom:8px;
  background:linear-gradient(90deg,#00d2ff,#7b2ff7);
  -webkit-background-clip:text;-webkit-text-fill-color:transparent}
.sub{color:#888;margin-bottom:32px;font-size:14px}
input{width:100%%;padding:14px 18px;border-radius:12px;border:1px solid rgba(255,255,255,0.15);
  background:rgba(255,255,255,0.05);color:#fff;font-size:16px;
  margin-bottom:16px;outline:none;transition:border .2s}
input:focus{border-color:#7b2ff7}
button{width:100%%;padding:14px;border-radius:12px;border:none;
  background:linear-gradient(135deg,#00d2ff,#7b2ff7);color:#fff;
  font-size:16px;font-weight:600;cursor:pointer;transition:opacity .2s}
button:hover{opacity:0.9}
button:disabled{opacity:0.5;cursor:not-allowed}
.result{margin-top:20px;padding:14px;border-radius:12px;font-size:14px;word-break:break-all}
.ok{background:rgba(0,210,100,0.15);border:1px solid rgba(0,210,100,0.3)}
.err{background:rgba(255,60,60,0.15);border:1px solid rgba(255,60,60,0.3)}
.info{margin-top:28px;color:#666;font-size:12px}
a{color:#00d2ff;text-decoration:none}
</style>
</head>
<body>
<div class="card">
  <h1>Axon Testnet Faucet</h1>
  <p class="sub">Get %s AXON test tokens · Cooldown: %s</p>
  <input id="addr" placeholder="Enter your 0x address" autocomplete="off" spellcheck="false">
  <button id="btn" onclick="drip()">Request Tokens</button>
  <div id="res"></div>
  <div class="info">
    <a href="/api/status" target="_blank">Faucet Status</a> ·
    Chain ID: 9001 ·
    <a href="https://github.com/Fatman2080/axon" target="_blank">GitHub</a>
  </div>
</div>
<script>
async function drip(){
  const addr=document.getElementById('addr').value.trim();
  const btn=document.getElementById('btn');
  const res=document.getElementById('res');
  if(!addr){res.className='result err';res.textContent='Please enter an address';return}
  btn.disabled=true;btn.textContent='Sending...';res.className='';res.textContent='';
  try{
    const r=await fetch('/api/faucet',{method:'POST',headers:{'Content-Type':'application/json'},
      body:JSON.stringify({address:addr})});
    const d=await r.json();
    if(d.success){
      res.className='result ok';
      res.innerHTML='✓ '+d.amount+' sent!<br>TX: <a href="#" style="color:#00d2ff">'+d.tx_hash+'</a>';
    }else{
      res.className='result err';res.textContent='✗ '+d.message;
    }
  }catch(e){res.className='result err';res.textContent='Network error: '+e.message}
  btn.disabled=false;btn.textContent='Request Tokens';
}
document.getElementById('addr').addEventListener('keydown',e=>{if(e.key==='Enter')drip()});
</script>
</body>
</html>`
