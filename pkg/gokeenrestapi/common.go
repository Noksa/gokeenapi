package gokeenrestapi

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/noksa/gokeenapi/internal/config"
	"github.com/noksa/gokeenapi/internal/keenspinner"
	"github.com/noksa/gokeenapi/pkg/models"
	"github.com/spf13/viper"
	"go.uber.org/multierr"
	"net/http"
	"strings"
)

var restyClient *resty.Client
var cookie string

func Auth() error {
	return keenspinner.WrapWithSpinner("Authorizing in API", func() error {
		response, err := GetApiClient().R().Get("/auth")
		var mErr error
		if response != nil {
			if response.StatusCode() == http.StatusUnauthorized {
				realm := response.Header().Get("x-ndm-realm")
				token := response.Header().Get("x-ndm-challenge")
				setCookieStr := response.Header().Get("set-cookie")
				setCookieStrSplitted := strings.Split(setCookieStr, ";")
				cookie = setCookieStrSplitted[0]
				secondRequest := GetApiClient().R()
				//secondRequest.Header.Set("Cookie", cookie)

				md5Hash := md5.New()
				md5Hash.Write([]byte(fmt.Sprintf("%v:%v:%v", viper.GetString(config.ViperKeeneticLogin), realm, viper.GetString(config.ViperKeeneticPassword))))
				md5HashArg := md5Hash.Sum(nil)
				md5HashStr := hex.EncodeToString(md5HashArg)
				sha256Hash := sha256.New()
				sha256Hash.Write([]byte(fmt.Sprintf("%v%v", token, md5HashStr)))
				sha256HashArg := sha256Hash.Sum(nil)
				sha256HashStr := hex.EncodeToString(sha256HashArg)
				secondRequest.SetBody(struct {
					Login    string `json:"login"`
					Password string `json:"password"`
				}{
					Login:    viper.GetString(config.ViperKeeneticLogin),
					Password: sha256HashStr,
				})
				response, err = secondRequest.Post("/auth")
				if err != nil {
					return err
				}
				if response.StatusCode() == http.StatusUnauthorized {
					return errors.New("can't authorize in keenetic. Verify your login and password")
				}
			}
		}
		mErr = multierr.Append(mErr, err)
		return mErr
	})
}

func ExecutePostParse(parse ...models.ParseRequest) ([]models.ParseResponse, error) {
	request := GetApiClient().R()
	request.SetBody(parse)
	response, err := request.Post("/rci/")
	var parseResponse []models.ParseResponse
	var mErr error
	if response != nil {
		if response.StatusCode() != http.StatusOK {
			return parseResponse, fmt.Errorf("wrong status code in response from api: %s", response.Status())
		}
		decodeErr := json.Unmarshal(response.Body(), &parseResponse)
		mErr = multierr.Append(mErr, decodeErr)
		for _, myParse := range parseResponse {
			for _, status := range myParse.Parse.Status {
				if status.Status == "error" {
					mErr = multierr.Append(mErr, fmt.Errorf("%s - %s - %s - %s", status.Status, status.Code, status.Ident, status.Message))
				}
			}
		}
	}
	mErr = multierr.Append(mErr, err)
	return parseResponse, mErr
}

func ExecuteGetSubPath(path string) ([]byte, error) {
	response, err := GetApiClient().R().Get(path)
	if err != nil {
		return nil, err
	}
	if response != nil {
		return response.Body(), nil
	}
	return []byte{}, errors.New("no response from keenetic api")
}

func GetApiClient() *resty.Client {
	if restyClient != nil {
		if restyClient.Header.Get("Cookie") == "" && cookie != "" {
			restyClient.Header.Set("Cookie", cookie)
		}
		return restyClient
	}
	restyClient = resty.New()
	restyClient.SetDisableWarn(true)
	restyClient.SetBaseURL(viper.GetString(config.ViperKeeneticUrl))
	return restyClient
}
