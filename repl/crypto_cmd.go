package main

import (
	"crypto/ed25519"
	"errors"
	"fmt"

	"github.com/gritzko/rdx"
)
import "encoding/hex"

func CmdSumSha256(ctx *REPL, arg *rdx.Iter) (ret []byte, err error) {
	if !arg.Read() {
		return nil, ErrBadArguments
	}
	subj := arg.Record()
	sha := rdx.Sha256Of(subj)
	ret = rdx.S0(hex.EncodeToString(sha[:]))
	return
}

func CmdCryptoKeyGen(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	_, sec, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, err
	}

	secHex := hex.EncodeToString(sec)

	out = rdx.S0(secHex)

	return out, nil
}

func CmdCryptoSign(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	if !args.Read() {
		return nil, ErrNoArgument
	}
	if args.Lit() != rdx.String {
		return nil, errors.New("first argument must be a ed25519 key in hex")
	}

	var privKey []byte
	privKeyHex := args.String()
	privKey, err = hex.DecodeString(privKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key hex: %w", err)
	}

	message := args.Rest()
	if len(message) == 0 {
		return nil, errors.New("missing data to sign")
	}

	signature := ed25519.Sign(privKey, message)

	out = rdx.P0(rdx.S0(hex.EncodeToString(signature)), message)

	return out, nil
}

func CmdCryptoVerify(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	// 1. Get key argument.
	if !args.Read() || args.Lit() != rdx.String {
		return nil, errors.New("missing key argument")
	}

	// 2. Extract public key.
	var pubKey []byte
	pubKey, err = hex.DecodeString(args.String())
	if len(pubKey) == ed25519.PrivateKeySize {
		pubKey = ed25519.PrivateKey(pubKey).Public().(ed25519.PublicKey)
	}
	if err != nil || pubKey == nil || len(pubKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key: %w", err)
	}

	// 3. Get the second argument: a tuple of (signature, message).
	if !args.Read() {
		return nil, errors.New("missing (signature, message) tuple")
	}
	if args.Lit() != rdx.Tuple {
		return nil, errors.New("second argument must be a tuple")
	}

	// 4. Extract signature and message from the inner tuple.
	innerIter := rdx.NewIter(args.Value())
	if !innerIter.Read() {
		return nil, errors.New("malformed inner tuple: missing signature")
	}
	sigHex := innerIter.String()
	signature, err := hex.DecodeString(sigHex)
	if err != nil || len(signature) != ed25519.SignatureSize {
		return nil, fmt.Errorf("invalid signature hex: %w", err)
	}

	message := innerIter.Rest()
	if len(message) == 0 {
		return nil, errors.New("malformed inner tuple: missing message")
	}

	// 5. Verify and return result as a term "OK" or nil
	isValid := ed25519.Verify(pubKey, message, signature)
	if isValid {
		out = rdx.T0("OK")
	} else {
		out = nil
	}

	return out, nil
}
