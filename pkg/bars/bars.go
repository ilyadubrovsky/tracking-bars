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

const (
	FormValueKeyUsername = "UserName"
	FormValueKeyPassword = "Password"
)

const (
	CookieNameAuthBars  = "auth_bars"
	CookieNameSessionID = "ASP.NET_SessionId"
)

var (
	ErrAuthorizationFailed = errors.New("authorization in BARS failed")
)

type Client interface {
	Authorization(ctx context.Context, username, password string) error
	MakeRequest(ctx context.Context, method string, url string, body io.Reader) (*http.Response, error)
	Clear()
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
	data := url.Values{
		FormValueKeyUsername: {username},
		FormValueKeyPassword: {password},
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
		return ErrAuthorizationFailed
	}

	return nil
}

// TODO кажется не универсальная штука, надо переделывать хедеры эти...
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

// TODO наверное, можно делать более эффективно
// Clear очищает данные внутри клиента, нужно делать перед каждой новой сессией
func (c *client) Clear() {
	jar, _ := cookiejar.New(nil)
	c.httpClient.Jar = jar
}

func (c *client) isAuthorized(response *http.Response) bool {
	cookies := response.Request.Cookies()

	for _, cookie := range cookies {
		if cookie.Name == CookieNameAuthBars || cookie.Name == CookieNameSessionID {
			return true
		}
	}

	return false
}
