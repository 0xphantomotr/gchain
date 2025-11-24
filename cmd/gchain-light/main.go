package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {
	rpcAddr := flag.String("rpc", "http://localhost:8000", "RPC endpoint base URL")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `gchain-light - lightweight RPC client

Usage:
  gchain-light [--rpc URL] <command> [args]

Commands:
  tip                           Show latest block height and hash.
  block <height>                Fetch block by height.
  balance <hex-address>         Show account balance.
  send --from A --to B --amount N  Submit a transaction.
`)
	}
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	cmd := args[0]
	cmdArgs := args[1:]

	client := &http.Client{Timeout: 5 * time.Second}

	switch cmd {
	case "tip":
		getAndPrint(client, fmt.Sprintf("%s/tip", *rpcAddr))
	case "block":
		if len(cmdArgs) != 1 {
			exitErr("block requires height argument")
		}
		if _, err := strconv.ParseUint(cmdArgs[0], 10, 64); err != nil {
			exitErr(fmt.Sprintf("invalid height: %v", err))
		}
		getAndPrint(client, fmt.Sprintf("%s/block/%s", *rpcAddr, cmdArgs[0]))
	case "balance":
		if len(cmdArgs) != 1 {
			exitErr("balance requires address argument")
		}
		getAndPrint(client, fmt.Sprintf("%s/balance/%s", *rpcAddr, cmdArgs[0]))
	case "send":
		sendFlags := flag.NewFlagSet("send", flag.ExitOnError)
		from := sendFlags.String("from", "", "hex sender address")
		to := sendFlags.String("to", "", "hex recipient address")
		amount := sendFlags.Uint64("amount", 0, "transfer amount")
		sendFlags.Parse(cmdArgs)

		if *from == "" || *to == "" || *amount == 0 {
			exitErr("send requires --from, --to, and --amount > 0")
		}

		payload := map[string]interface{}{
			"from":   *from,
			"to":     *to,
			"amount": *amount,
		}
		data, _ := json.Marshal(payload)
		req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/tx", *rpcAddr), bytes.NewReader(data))
		if err != nil {
			exitErr(err.Error())
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			exitErr(err.Error())
		}
		printResponse(resp)
	default:
		exitErr(fmt.Sprintf("unknown command %q", cmd))
	}
}

func getAndPrint(client *http.Client, url string) {
	resp, err := client.Get(url)
	if err != nil {
		exitErr(err.Error())
	}
	printResponse(resp)
}

func printResponse(resp *http.Response) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		exitErr(err.Error())
	}
	if resp.StatusCode >= 400 {
		exitErr(fmt.Sprintf("server error: %s", body))
	}
	fmt.Println(string(body))
}

func exitErr(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
