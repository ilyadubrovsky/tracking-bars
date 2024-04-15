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

// TODO ErrWrongGradesPage
var ErrNoAuth = errors.New("authorization in BARS has not been completed")

type Client interface {
	Authorization(ctx context.Context, username, password string) error
	MakeRequest(ctx context.Context, method string, url string, body io.Reader) (*http.Response, error)
	Clear(ctx context.Context)
}

type client struct {
	httpClient      *http.Client
	registrationURL string
}

func NewClient(registrationURL string) Client {
	jar, _ := cookiejar.New(nil)
	return &client{
		httpClient:      &http.Client{Jar: jar},
		registrationURL: registrationURL,
	}
}

func (c *client) Authorization(ctx context.Context, username, password string) error {
	verificationToken, err := c.getVerificationToken(ctx)
	if err != nil {
		return fmt.Errorf("getVerificationToken: %w", err)
	}

	data := url.Values{
		"__RequestVerificationToken": {verificationToken},
		"UserName":                   {username},
		"Password":                   {password},
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.registrationURL,
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return fmt.Errorf("http.NewRequestWithContext: %w", err)
	}

	request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 YaBrowser/22.11.5.715 Yowser/2.5 Safari/537.36")
	request.Header.Set("Accept-Language", "ru,en;q=0.9")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("httpClient.Do: %w", err)
	}

	_, err = io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("io.ReadAll (response.Body): %w", err)
	}

	defer response.Body.Close()

	if !c.isAuthorized(response) {
		return ErrNoAuth
	}

	return nil
}

func (c *client) MakeRequest(ctx context.Context, method string, url string, body io.Reader) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("http.NewRequestWithContext: %w", err)
	}

	request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 YaBrowser/22.11.5.715 Yowser/2.5 Safari/537.36")
	request.Header.Set("Accept-Language", "ru,en;q=0.9")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("httpClient.Do: %w", err)
	}

	return response, nil
}

// Clear очищает данные внутри клиента, нужно делать перед каждой новой сессией
// TODO наверное, можно делать более эффективно
func (c *client) Clear(ctx context.Context) {
	jar, _ := cookiejar.New(nil)
	c.httpClient.Jar = jar
}

func (c *client) getVerificationToken(ctx context.Context) (string, error) {
	response, err := c.MakeRequest(ctx, http.MethodPost, c.registrationURL, nil)
	if err != nil {
		return "", fmt.Errorf("MakeRequest: %w", err)
	}

	cookies := response.Cookies()

	for _, cookie := range cookies {
		if cookie.Name == "__RequestVerificationToken_L2JhcnNfd2Vi0" {
			return cookie.Value, nil
		}
	}

	return "", errors.New("verification token was not provided by BARS")
}

func (c *client) isAuthorized(response *http.Response) bool {
	cookies := response.Request.Cookies()

	for _, cookie := range cookies {
		if cookie.Name == "auth_bars" || cookie.Name == "ASP.NET_SessionId" {
			return true
		}
	}

	return false
}
