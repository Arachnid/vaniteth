package main

import (
    "bytes"
    "crypto/ecdsa"
    "crypto/rand"
    "encoding/hex"
    "flag"
    "fmt"
    "os"

    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/crypto"
    "github.com/ethereum/go-ethereum/crypto/secp256k1"
)

// Signature for a function which returns true iff a is 'better' than b
type betterAddressFunc func(a common.Address, b common.Address) bool

type Result struct {
    address    common.Address
    privateKey *ecdsa.PrivateKey
    nonce      int
}

func leastScorer(a, b common.Address) bool {
    return bytes.Compare(a.Bytes(), b.Bytes()) < 0
}

func mostScorer(a, b common.Address) bool {
    return bytes.Compare(a.Bytes(), b.Bytes()) > 0
}

func ascendingScorer(a, b common.Address) bool {
    result := countAscending(a.Bytes(), false) - countAscending(b.Bytes(), false)

    if result > 0 {
        return true
    } else if result < 0 {
        return false
    } else {
        return leastScorer(a, b)
    }
}

func strictAscendingScorer(a, b common.Address) bool {
    result := countAscending(a.Bytes(), true) - countAscending(b.Bytes(), true)

    if result > 0 {
        return true
    } else if result < 0 {
        return false
    } else {
        return leastScorer(a, b)
    }
}

func countAscending(data []byte, strict bool) int {
    count := 0
    var last byte = 0
    for i := 0; i < 20; i++ {
        for j := 4; j >= 0; j -= 4 {
            nybble := (data[i] >> uint(j)) & 0xf
            if nybble < last || (nybble > last + 1 && strict) {
                return count
            }
            last = nybble
            count += 1
        }
    }
    return 40 // as if
}

var (
    threads = flag.Int("threads", 2, "Number of threads to run")
    contractAddress = flag.Bool("contract", false, "Derive addresses for deployed contracts instead of accounts")
    maxNonce = flag.Int("maxnonce", 32, "Maximum nonce value to test when deriving contract addresses")
    scorer = flag.String("scorer", "least", "Scoring function to use. Options include 'least'")

    scorers = map[string]betterAddressFunc{
        "least":            leastScorer,
        "most":             mostScorer,
        "ascending":        ascendingScorer,
        "strictAscending":  strictAscendingScorer,
    }
)

func main() {
    flag.Parse()

    scoreFunc, ok := scorers[*scorer]
    if !ok {
        fmt.Printf("Invalid score function '%s'\n", *scorer)
        os.Exit(1);
    }

    results := make(chan Result)
    for i := 0; i < *threads; i++ {
        go start(results, *contractAddress, *maxNonce, scoreFunc)
    }

    best := <-results
    for next := range results {
        if scoreFunc(next.address, best.address) {
            best = next
            if *contractAddress {
                fmt.Printf("%s\t%d\t%s\n", best.address.Hex(), best.nonce, hex.EncodeToString(crypto.FromECDSA(best.privateKey)))
            } else {
                fmt.Printf("%s\t%d\t%s\n", best.address.Hex(), best.nonce, hex.EncodeToString(crypto.FromECDSA(best.privateKey)))
            }
        }
    }
}

func start(results chan<- Result, contracts bool, maxNonce int, scoreFunc betterAddressFunc) {
    addresses := make(chan Result)
    go generateAddresses(addresses, contracts, maxNonce)

    best := <-addresses
    results <- best
    for next := range addresses {
        if scoreFunc(next.address, best.address) {
            best = next
            results <- next
        }
    }
}

func generateAddresses(out chan<- Result, contracts bool, maxNonce int) {
    for {
        privateKey, err := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
        if err != nil {
            fmt.Printf("Error generating ECDSA keypair: %v\n", err)
            os.Exit(1)
        }

        contractAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
        if contracts {
            for i := 0; i < maxNonce; i++ {
                address := crypto.CreateAddress(contractAddress, uint64(i))
                out <- Result{address, privateKey, i}
            }
        } else {
            out <- Result{contractAddress, privateKey, 0}
        }
    }
    os.Exit(0)
}
