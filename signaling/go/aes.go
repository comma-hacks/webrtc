// https://gist.github.com/locked/b066aa1ddeb2b28e855e
// https://gist.github.com/yingray/57fdc3264b1927ef0f984b533d63abab
package secureput

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
)

func encrypt(key []byte, plainText []byte) (encoded string, err error) {
	//Create a new AES cipher using the key
	block, err := aes.NewCipher(key)

	//IF NewCipher failed, exit:
	if err != nil {
		return
	}

	iv := RandSeq(aes.BlockSize)

	plainText = append([]byte(iv), plainText...)

	bPlaintext := PKCS7Padding(plainText, aes.BlockSize, len(plainText))

	cipherText := make([]byte, len(bPlaintext))

	cbc := cipher.NewCBCEncrypter(block, []byte(iv))
	cbc.CryptBlocks(cipherText, bPlaintext)

	//Return string encoded in base64
	return base64.StdEncoding.EncodeToString(cipherText), err
}

func decrypt(key []byte, secure string) (decoded []byte, err error) {
	//Remove base64 encoding:
	cipherText, err := base64.StdEncoding.DecodeString(secure)

	//IF DecodeString failed, exit:
	if err != nil {
		return
	}

	//Create a new AES cipher with the key and encrypted message
	block, err := aes.NewCipher(key)

	//IF NewCipher failed, exit:
	if err != nil {
		return
	}

	//IF the length of the cipherText is less than 16 Bytes:
	if len(cipherText) < aes.BlockSize {
		err = errors.New("ciphertext block size is too short")
		return
	}

	iv := cipherText[:aes.BlockSize]
	cipherText = cipherText[aes.BlockSize:]

	//Decrypt the message
	// stream := cipher.NewCFBDecrypter(block, iv)
	// stream.XORKeyStream(cipherText, cipherText)
	// crypto.AES.

	cbc := cipher.NewCBCDecrypter(block, iv)
	cbc.CryptBlocks(cipherText, cipherText)

	return cipherText, err
}

func PKCS7Padding(ciphertext []byte, blockSize int, after int) []byte {
	padding := (blockSize - len(ciphertext)%blockSize)
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}
