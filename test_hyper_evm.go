package main

import (
"bytes"
"encoding/json"
"fmt"
"io"
"net/http"
"math/big"
"strings"
)

func main() {
	rpcURL := "https://rpc.hyperliquid.xyz/evm"
	addr := "0x2Aa4608Be9772fe32D83F6b8eDdbEf343eE522B2"
	
	cleanAddr := strings.TrimPrefix(addr, "0x")
	paddedAddr := fmt.Sprintf("%064s", cleanAddr)
	
	// USDC address provided by user
	usdcAddr := "0xb88339CB7199b77E23DB6E890353E22632Ba630f"
	
	balReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_call",
		"params": []interface{}{
			map[string]string{
				"to":   usdcAddr,
				"data": "0x70a08231" + paddedAddr,
			},
			"latest",
		},
		"id": 1,
	}
	bBal, _ := json.Marshal(balReq)
	respBal, _ := http.Post(rpcURL, "application/json", bytes.NewBuffer(bBal))
	bodyBal, _ := io.ReadAll(respBal.Body)
	var resBal struct{ Result string `json:"result"` }
	json.Unmarshal(bodyBal, &resBal)
	
	fmt.Println("Raw USDC balance result:", string(bodyBal))
	
	if resBal.Result != "" && resBal.Result != "0x" {
		balInt := new(big.Int)
		balInt.SetString(strings.TrimPrefix(resBal.Result, "0x"), 16)
		balFloat, _ := new(big.Float).SetInt(balInt).Float64()
		fmt.Printf("USDC Balance of %s: %f\n", addr, balFloat/1e6)
	}
}
