package pgp

import (
	"bytes"
	"crypto"
	"crypto/dsa"
	"crypto/ecdsa"
	"crypto/rsa"
	"hash"
	"io"
	"strconv"
	"time"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/errors"
	"golang.org/x/crypto/openpgp/packet"
)

// hashForSignature returns a pair of hashes that can be used to verify a
// signature. The signature may specify that the contents of the signed message
// should be preprocessed (i.e. to normalize line endings). Thus this function
// returns two hashes. The second should be used to hash the message itself and
// performs any needed preprocessing.
func hashForSignature(hashID crypto.Hash, sigType packet.SignatureType) (hash.Hash, hash.Hash, error) {
	if !hashID.Available() {
		return nil, nil, errors.UnsupportedError("hash not available: " + strconv.Itoa(int(hashID)))
	}
	h := hashID.New()

	switch sigType {
	case packet.SigTypeBinary:
		return h, h, nil
	case packet.SigTypeText:
		return h, openpgp.NewCanonicalTextHash(h), nil
	}

	return nil, nil, errors.UnsupportedError("unsupported signature type: " + strconv.Itoa(int(sigType)))
}

type signatureResult struct {
	CreationTime time.Time
	IssuerKeyID  uint64
	PubKeyAlgo   packet.PublicKeyAlgorithm
	Entity       *openpgp.Entity
}

// checkDetachedSignature takes a signed file and a detached signature and
// returns the signer if the signature is valid. If the signer isn't known,
// ErrUnknownIssuer is returned.
//
// This is extended version of golang.org/x/crypto/openpgp to support multiple signers and returns multiple
// signers, plus keeps track of "missing" keys
func checkDetachedSignature(keyring openpgp.KeyRing, signed, signature io.Reader) (signers []signatureResult, missingKeys int, err error) {
	var p packet.Packet

	signedBuf := &bytes.Buffer{}
	if _, e := io.Copy(signedBuf, signed); e != nil && e != io.EOF {
		return nil, 0, e
	}

	packets := packet.NewReader(signature)
	for {
		p, err = packets.Next()
		if err == io.EOF {
			if len(signers) == 0 || missingKeys > 0 {
				err = errors.ErrUnknownIssuer
			} else {
				err = nil
			}

			return
		}
		if err != nil {
			return nil, 0, err
		}

		var issuerKeyID uint64
		var hashFunc crypto.Hash
		var sigType packet.SignatureType
		var creationTime time.Time
		var pubKeyAlgo packet.PublicKeyAlgorithm
		var keys []openpgp.Key

		switch sig := p.(type) {
		case *packet.Signature:
			if sig.IssuerKeyId == nil {
				return nil, 0, errors.StructuralError("signature doesn't have an issuer")
			}
			issuerKeyID = *sig.IssuerKeyId
			hashFunc = sig.Hash
			sigType = sig.SigType
			creationTime = sig.CreationTime
			pubKeyAlgo = sig.PubKeyAlgo
		case *packet.SignatureV3:
			issuerKeyID = sig.IssuerKeyId
			hashFunc = sig.Hash
			sigType = sig.SigType
			creationTime = sig.CreationTime
			pubKeyAlgo = sig.PubKeyAlgo
		default:
			return nil, 0, errors.StructuralError("non signature packet found")
		}

		keys = keyring.KeysByIdUsage(issuerKeyID, packet.KeyFlagSign)
		if len(keys) == 0 {
			signers = append(signers, signatureResult{
				CreationTime: creationTime,
				IssuerKeyID:  issuerKeyID,
				PubKeyAlgo:   pubKeyAlgo,
			})
			missingKeys++
			continue
		}

		h, wrappedHash, err := hashForSignature(hashFunc, sigType)
		if err != nil {
			return nil, 0, err
		}

		if _, e := io.Copy(wrappedHash, bytes.NewReader(signedBuf.Bytes())); e != nil && e != io.EOF {
			return nil, 0, e
		}

		allFailed := true
		for _, key := range keys {
			switch sig := p.(type) {
			case *packet.Signature:
				err = key.PublicKey.VerifySignature(h, sig)
			case *packet.SignatureV3:
				err = key.PublicKey.VerifySignatureV3(h, sig)
			default:
				panic("unreachable")
			}

			if err == nil {
				signers = append(signers, signatureResult{
					CreationTime: creationTime,
					IssuerKeyID:  issuerKeyID,
					PubKeyAlgo:   pubKeyAlgo,
					Entity:       key.Entity,
				})
				allFailed = false
			}
		}

		if allFailed {
			return nil, 0, err
		}
	}
}

// readArmored reads an armored block with the given type.
func readArmored(r io.Reader, expectedType string) (body io.Reader, err error) {
	block, err := armor.Decode(r)
	if err != nil {
		return
	}

	if block.Type != expectedType {
		return nil, errors.InvalidArgumentError("expected '" + expectedType + "', got: " + block.Type)
	}

	return block.Body, nil
}

// checkArmoredDetachedSignature performs the same actions as
// CheckDetachedSignature but expects the signature to be armored.
//
// This is extended version of golang.org/x/crypto/openpgp to support multiple signers and returns multiple
// signers, plus keeps track of "missing" keys
func checkArmoredDetachedSignature(keyring openpgp.KeyRing, signed, signature io.Reader) (signers []signatureResult, missingKeys int, err error) {
	body, err := readArmored(signature, openpgp.SignatureType)
	if err != nil {
		return
	}

	return checkDetachedSignature(keyring, signed, body)
}

func pubkeyAlgorithmName(algorithm packet.PublicKeyAlgorithm) string {
	switch algorithm {
	case packet.PubKeyAlgoRSA, packet.PubKeyAlgoRSAEncryptOnly, packet.PubKeyAlgoRSASignOnly:
		return "RSA"
	case packet.PubKeyAlgoElGamal:
		return "ElGamal"
	case packet.PubKeyAlgoDSA:
		return "DSA"
	case packet.PubKeyAlgoECDH:
		return "EDCH"
	case packet.PubKeyAlgoECDSA:
		return "ECDSA"
	}

	return "unknown"
}

func keyBits(key interface{}) string {
	switch k := key.(type) {
	case *rsa.PublicKey:
		return strconv.Itoa(k.N.BitLen())
	case *dsa.PublicKey:
		return strconv.Itoa(k.P.BitLen())
	case *ecdsa.PublicKey:
		return strconv.Itoa(k.Curve.Params().BitSize)
	default:
		return "?"
	}
}

func validEntity(entity *openpgp.Entity) bool {
	var selfSig *packet.Signature
	for _, ident := range entity.Identities {
		if selfSig == nil {
			selfSig = ident.SelfSignature
		} else if ident.SelfSignature.IsPrimaryId != nil && *ident.SelfSignature.IsPrimaryId {
			selfSig = ident.SelfSignature
			break
		}
	}

	if selfSig == nil {
		return false
	}

	if len(entity.Revocations) > 0 {
		return false
	}

	if selfSig.RevocationReason != nil {
		return false
	}

	if !selfSig.FlagsValid {
		return false
	}

	if selfSig.KeyLifetimeSecs != nil && selfSig.CreationTime.Add(time.Duration(*selfSig.KeyLifetimeSecs)*time.Second).Before(time.Now()) {
		return false
	}

	return true
}
