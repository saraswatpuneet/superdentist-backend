package helpers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/superdentist/superdentist-backend/global"
)

// ResponseError ... essentially a single point of sending some error to route back
func ResponseError(w http.ResponseWriter, httpStatusCode int, err error) {
	log.Errorf("Response error %s", err.Error())
	response, _ := json.Marshal(err)
	w.Header().Add("Status", strconv.Itoa(httpStatusCode)+" "+err.Error())
	w.Header().Add("content-type", "application/json")
	w.WriteHeader(httpStatusCode)

	if _, err := w.Write(response); err != nil {
		log.Errorf("ResponseError ... unable to write JSON response: %v", err)
	}
}

func EncryptAndEncode(toencode string) (string, error) {
	nonce := make([]byte, global.Options.GCMQR.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := global.Options.GCMQR.Seal(nil, nonce, []byte(toencode), nil)
	str := base64.URLEncoding.EncodeToString(append(nonce, ciphertext...))
	return str, nil
}

func DecryptAndDecode(ciphertext64 string) (string, error) {
	ciphertext, err := base64.URLEncoding.DecodeString(ciphertext64)
	if err != nil {
		return "", err
	}

	nonceSize := global.Options.GCMQR.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	b, err := global.Options.GCMQR.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(b), err
}
