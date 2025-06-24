package main

import (
	"flag"
	"fmt"
	"github.com/cwkr/auth-server/internal/oauth2/keyset"
	"log"
	"os"
)

func main() {
	var (
		outFilename string
		keySize     int
		keyID       string
		err         error
		keyBytes    []byte
	)

	log.SetOutput(os.Stdout)

	flag.StringVar(&outFilename, "o", "", "output file")
	flag.IntVar(&keySize, "size", 2048, "key size")
	flag.StringVar(&keyID, "id", "", "key id")
	flag.Parse()

	if keySize < 1024 {
		panic("key size less than 1024")
	}

	keyBytes, err = keyset.GeneratePrivateKey(keySize, keyID)
	if err != nil {
		panic(err)
	}

	if outFilename == "" {
		fmt.Print(string(keyBytes))
	} else {
		err := os.WriteFile(outFilename, keyBytes, 0600)
		if err != nil {
			panic(err)
		}
	}
}
