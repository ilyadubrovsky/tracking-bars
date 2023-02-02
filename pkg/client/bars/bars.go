package bars

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

var ErrNoAuth = errors.New("authorization in BARS hasnt been completed")

type Client interface {
	GetPage(ctx context.Context, method string, url string, body io.Reader) (*http.Response, error)
	Authorization(ctx context.Context, username, password string) error
}

type client struct {
	HttpClient      *http.Client
	registrationURL string
}

func NewClient(registrationURL string) *client {
	jar, _ := cookiejar.New(nil)
	return &client{
		HttpClient:      &http.Client{Jar: jar},
		registrationURL: registrationURL,
	}
}

func (client *client) Authorization(ctx context.Context, username, password string) error {
	cl := client.HttpClient
	verificationToken, err := client.getVerificationToken(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get a verification token due error: %s", err)
	}
	remember := "true"

	data := url.Values{
		"__RequestVerificationToken": {verificationToken},
		"UserName":                   {username},
		"Password":                   {password},
		"Remember":                   {remember},
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.registrationURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to POST-request due error: %s", err)
	}

	request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 YaBrowser/22.11.5.715 Yowser/2.5 Safari/537.36")
	request.Header.Set("Accept-Language", "ru,en;q=0.9")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := cl.Do(request)
	if err != nil {
		return fmt.Errorf("failed to POST-request due error: %s", err)

	}

	defer response.Body.Close()

	authstatus, err := client.authStatus(response)
	if err != nil {
		return err
	}

	_, err = ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read a POST-response due error: %s", err)
	}

	if authstatus == false {
		return ErrNoAuth
	}
	return nil
}

func (client *client) GetPage(ctx context.Context, method string, url string, body io.Reader) (*http.Response, error) {
	c := client.HttpClient

	request, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to %s-request due error: %s", method, err)
	}

	request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 YaBrowser/22.11.5.715 Yowser/2.5 Safari/537.36")
	request.Header.Set("Accept-Language", "ru,en;q=0.9")

	response, err := c.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to %s-request due error: %s", method, err)
	}

	return response, nil
}

func (client *client) getVerificationToken(ctx context.Context) (string, error) {
	response, err := client.GetPage(ctx, http.MethodPost, client.registrationURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get a page (getVerificationToken) due error: %s", err)
	}

	cookies := response.Cookies()

	for _, cookie := range cookies {
		if cookie.Name == "__RequestVerificationToken_L2JhcnNfd2Vi0" {
			return cookie.Value, nil
		}
	}

	return "", errors.New("failed to get a verification token")
}

func (client *client) authStatus(response *http.Response) (bool, error) {
	cookies := response.Request.Cookies()

	for _, cookie := range cookies {
		if cookie.Name == "auth_bars" {
			return true, nil
		}
	}

	return false, ErrNoAuth
}
