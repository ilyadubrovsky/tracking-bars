package aes

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
)

func EncryptCFB(key, decryptedData []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	ciphertext := make([]byte, aes.BlockSize+len(decryptedData))
	iv := ciphertext[:aes.BlockSize]
	if _, err = rand.Read(iv); err != nil {
		return nil, err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], decryptedData)

	return ciphertext, nil
}

func DecryptCFB(key, encryptedData []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	iv := encryptedData[:aes.BlockSize]

	stream := cipher.NewCFBDecrypter(block, iv)

	plaintext := make([]byte, len(encryptedData))
	stream.XORKeyStream(plaintext, encryptedData)

	return string(plaintext[aes.BlockSize:]), nil
}
