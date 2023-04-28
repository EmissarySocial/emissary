package publicKey

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
)

func EncodePrivatePEM(privateKey *rsa.PrivateKey) string {

	// Get ASN.1 DER format
	privDER := x509.MarshalPKCS1PrivateKey(privateKey)

	// pem.Block
	privBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privDER,
	}

	// Private key in PEM format
	privatePEM := pem.EncodeToMemory(&privBlock)

	return string(privatePEM)
}

func EncodePublicPEM(privateKey *rsa.PrivateKey) string {

	// Get ASN.1 DER format
	publicDER := x509.MarshalPKCS1PublicKey(&privateKey.PublicKey)

	// pem.Block
	publicBlock := pem.Block{
		Type:    "RSA PUBLIC KEY",
		Headers: nil,
		Bytes:   publicDER,
	}

	// Private key in PEM format
	publicPEM := pem.EncodeToMemory(&publicBlock)

	return string(publicPEM)
}
