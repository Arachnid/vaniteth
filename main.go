package main

import (
    "crypto/ecdsa"
    "crypto/rand"
    "encoding/hex"
    "flag"
    "fmt"
    "os"
    "strings"

    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/crypto"
    "github.com/ethereum/go-ethereum/crypto/secp256k1"
)

// Signature for a function which returns >0 if a>b, <0 if a<b, and 0 otherwise
type addressComparer func(a common.Address, b common.Address) int

type Result struct {
    address    common.Address
    privateKey *ecdsa.PrivateKey
    nonce      int
}

func targetScorer(a, b common.Address) int {
    return countPrefix(a.Bytes(), target.Bytes()) - countPrefix(b.Bytes(), target.Bytes())
}

func countPrefix(a, b []byte) int {
    for i := 0; i < 20; i++ {
        for j := 0; j <= 1; j++ {
            shift := 4 * (1 - uint(j))
            if (a[i] >> shift) & 0xf != (b[i] >> shift) & 0xf {
                return i * 2 + j
            }
        }
    }
    return 40
}

func ascendingScorer(a, b common.Address) int {
    return countAscending(a.Bytes(), false) - countAscending(b.Bytes(), false)
}

func strictAscendingScorer(a, b common.Address) int {
    return countAscending(a.Bytes(), true) - countAscending(b.Bytes(), true)
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

type StringList []string

func (sl *StringList) String() string {
    return strings.Join([]string(*sl), ",")
}

func (sl *StringList) Set(value string) error {
    parts := strings.Split(value, ",")
    *sl = make([]string, len(parts))
    copy(*sl, parts)
    return nil
}

var (
    threads = flag.Int("threads", 2, "Number of threads to run")
    contractAddress = flag.Bool("contract", false, "Derive addresses for deployed contracts instead of accounts")
    maxNonce = flag.Int("maxnonce", 32, "Maximum nonce value to test when deriving contract addresses")
    targetHex = flag.String("target", "0x0000000000000000000000000000000000000000", "Target address to mine towards")
    target common.Address
    scorers = StringList{"target", "ascending", "strictAscending"}

    scoreFuncs = map[string]addressComparer{
        "ascending":        ascendingScorer,
        "strictAscending":  strictAscendingScorer,
        "target":           targetScorer,
    }
)

func scoreTest(funcs map[string]addressComparer, bests map[string]common.Address, a common.Address) (better bool) {
    for name, scoreFunc := range funcs {
        best, ok := bests[name]
        if !ok || scoreFunc(a, best) >= 0 {
            better = true
            bests[name] = a
        }
    }
    return better
}

func main() {
    flag.Var(&scorers, "scorers", "List of score functions to use")
    flag.Parse()
    target = common.HexToAddress(*targetHex)

    funcs := make(map[string]addressComparer)
    for _, k := range scorers {
        funcs[k] = scoreFuncs[k]
    }

    results := make(chan Result)
    for i := 0; i < *threads; i++ {
        go start(results, *contractAddress, *maxNonce, funcs)
    }

    bests := make(map[string]common.Address)
    for next := range results {
        if scoreTest(funcs, bests, next.address) {
            if *contractAddress {
                fmt.Printf("%s\t%d\t%s\n", next.address.Hex(), next.nonce, hex.EncodeToString(crypto.FromECDSA(next.privateKey)))
            } else {
                fmt.Printf("%s\t%d\t%s\n", next.address.Hex(), next.nonce, hex.EncodeToString(crypto.FromECDSA(next.privateKey)))
            }
        }
    }
}

func start(results chan<- Result, contracts bool, maxNonce int, funcs map[string]addressComparer) {
    addresses := make(chan Result)
    go generateAddresses(addresses, contracts, maxNonce)

    bests := make(map[string]common.Address)
    for next := range addresses {
        if scoreTest(funcs, bests, next.address) {
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
