package bars

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

var ErrNoAuth = errors.New("authorization in BARS hasnt been completed")

type Client struct {
	HttpClient      *http.Client
	registrationURL string
}

func NewClient(registrationURL string) *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		HttpClient:      &http.Client{Jar: jar},
		registrationURL: registrationURL,
	}
}

func (client *Client) Authorization(ctx context.Context, username, password string) error {
	cl := client.HttpClient
	verificationToken, err := client.getVerificationToken(ctx)
	if err != nil {
		return fmt.Errorf("client authorization: %v", err)
	}

	data := url.Values{
		"__RequestVerificationToken": {verificationToken},
		"UserName":                   {username},
		"Password":                   {password},
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.registrationURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 YaBrowser/22.11.5.715 Yowser/2.5 Safari/537.36")
	request.Header.Set("Accept-Language", "ru,en;q=0.9")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := cl.Do(request)
	if err != nil {
		return err
	}

	_, err = io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	authstatus := client.authStatus(response)

	if !authstatus {
		return ErrNoAuth
	}

	return nil
}

func (client *Client) GetPage(ctx context.Context, method string, url string, body io.Reader) (*http.Response, error) {
	c := client.HttpClient

	request, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 YaBrowser/22.11.5.715 Yowser/2.5 Safari/537.36")
	request.Header.Set("Accept-Language", "ru,en;q=0.9")

	response, err := c.Do(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (client *Client) getVerificationToken(ctx context.Context) (string, error) {
	response, err := client.GetPage(ctx, http.MethodPost, client.registrationURL, nil)
	if err != nil {
		return "", fmt.Errorf("registration page was not received: %v", err)
	}

	cookies := response.Cookies()

	for _, cookie := range cookies {
		if cookie.Name == "__RequestVerificationToken_L2JhcnNfd2Vi0" {
			return cookie.Value, nil
		}
	}

	return "", fmt.Errorf("verification token was not provided by BARS")
}

func (client *Client) authStatus(response *http.Response) bool {
	cookies := response.Request.Cookies()

	for _, cookie := range cookies {
		if cookie.Name == "auth_bars" || cookie.Name == "ASP.NET_SessionId=k5ze3r11df3absu0idsy2xj5" {
			return true
		}
	}

	return false
}
