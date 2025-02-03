package stages

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	authenticateURL = "https://auth.sberclass.ru/auth/realms/EduPowerKeycloak/login-actions/authenticate"
)

var (
	ErrWrongRedirect = errors.New("the redirect was other than what was expected")
	ErrCredentials   = errors.New("user credentials were wrong")
)

func Authenticate(ctx context.Context, preauth PreauthResult, login, password string) error {
	gotTheRedirect := false
	client := http.Client{
		CheckRedirect: createCheckRedirect(&gotTheRedirect),
	}

	data := url.Values{}
	data.Set("username", login)
	data.Set("password", password)

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, authenticateURL,
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	request.AddCookie(&http.Cookie{Name: "AUTH_SESSION_ID", Value: preauth.authSessionID})
	request.AddCookie(&http.Cookie{Name: "AUTH_SESSION_ID_LEGACY", Value: preauth.authSessionID})
	request.AddCookie(&http.Cookie{Name: "KC_RESTART", Value: preauth.kcRestart})

	query := request.URL.Query()

	query.Set(sessionCodeTag, preauth.sessionCode)
	query.Set(executionTag, preauth.execution)
	query.Set("client_id", "school21")
	query.Set(tabIDtag, preauth.tabID)

	request.URL.RawQuery = query.Encode()

	resp, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}

	defer resp.Body.Close()

	if !gotTheRedirect { // no redirect = wrong creds
		return ErrCredentials
	}

	return nil
}

func createCheckRedirect(gotTheRedirect *bool) func(req *http.Request, via []*http.Request) error {
	return func(req *http.Request, _ []*http.Request) error {
		if req.URL.Path == "/auth/realms/EduPowerKeycloak/login-actions/required-action" {
			*gotTheRedirect = true

			return http.ErrUseLastResponse
		}

		return ErrWrongRedirect
	}
}
