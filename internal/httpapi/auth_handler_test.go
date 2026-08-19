package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestIsSecureBehindHTTPSProxy(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://snipvault.test/", nil)
	request.Header.Set("X-Forwarded-Proto", "https")
	if !requestIsSecure(request) {
		t.Fatal("expected forwarded HTTPS request to be secure")
	}
}

func TestAccountSecurityEndpoints(t *testing.T) {
	handler, sender := testHandlerWithMailer()
	register := httptest.NewRecorder()
	handler.ServeHTTP(register, httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"username":"owner","email":"owner@example.com","password":"password123"}`)))

	unverified := httptest.NewRecorder()
	handler.ServeHTTP(unverified, httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"owner@example.com","password":"password123"}`)))
	if unverified.Code != http.StatusForbidden {
		t.Fatalf("expected unverified login to fail, got %d", unverified.Code)
	}

	verify := httptest.NewRecorder()
	handler.ServeHTTP(verify, httptest.NewRequest(http.MethodPost, "/verify-email", strings.NewReader(`{"email":"owner@example.com","code":"`+sender.verificationCode+`"}`)))
	if verify.Code != http.StatusOK {
		t.Fatalf("verification failed: %s", verify.Body.String())
	}
	var session struct {
		CSRFToken string `json:"csrf_token"`
	}
	_ = json.Unmarshal(verify.Body.Bytes(), &session)
	cookies := verify.Result().Cookies()

	request := func(method, path, body string, protected bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
		if protected {
			req.Header.Set("X-CSRF-Token", session.CSRFToken)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		return response
	}
	if response := request(http.MethodGet, "/me/profile", "", false); response.Code != http.StatusOK {
		t.Fatalf("profile failed: %s", response.Body.String())
	}
	if response := request(http.MethodPatch, "/me/profile", `{"username":"renamed"}`, true); response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "renamed") {
		t.Fatalf("profile update failed: %s", response.Body.String())
	}
	if response := request(http.MethodPost, "/me/password", `{"current_password":"password123","new_password":"changed123"}`, true); response.Code != http.StatusNoContent {
		t.Fatalf("password change failed: %s", response.Body.String())
	}

	forgot := httptest.NewRecorder()
	handler.ServeHTTP(forgot, httptest.NewRequest(http.MethodPost, "/forgot-password", strings.NewReader(`{"email":"owner@example.com"}`)))
	if forgot.Code != http.StatusAccepted || sender.resetCode == "" {
		t.Fatal("forgot password failed")
	}
	reset := httptest.NewRecorder()
	handler.ServeHTTP(reset, httptest.NewRequest(http.MethodPost, "/reset-password", strings.NewReader(`{"email":"owner@example.com","code":"`+sender.resetCode+`","password":"reset1234"}`)))
	if reset.Code != http.StatusNoContent {
		t.Fatalf("reset failed: %s", reset.Body.String())
	}

	login := httptest.NewRecorder()
	handler.ServeHTTP(login, httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"owner@example.com","password":"reset1234"}`)))
	if login.Code != http.StatusOK {
		t.Fatalf("reset login failed: %s", login.Body.String())
	}
	_ = json.Unmarshal(login.Body.Bytes(), &session)
	cookies = login.Result().Cookies()
	if response := request(http.MethodDelete, "/me/account", `{"password":"reset1234"}`, true); response.Code != http.StatusNoContent {
		t.Fatalf("delete failed: %s", response.Body.String())
	}
}

func TestVerificationValidationAndResend(t *testing.T) {
	handler, _ := testHandlerWithMailer()
	bad := httptest.NewRecorder()
	handler.ServeHTTP(bad, httptest.NewRequest(http.MethodPost, "/verify-email", strings.NewReader(`{"email":"x@example.com","code":"12"}`)))
	if bad.Code != http.StatusBadRequest {
		t.Fatal("short verification code must fail")
	}
	resend := httptest.NewRecorder()
	handler.ServeHTTP(resend, httptest.NewRequest(http.MethodPost, "/resend-verification", strings.NewReader(`{"email":"missing@example.com"}`)))
	if resend.Code != http.StatusAccepted {
		t.Fatal("resend response must not reveal account existence")
	}
}

func TestRegisterAndLogin(t *testing.T) {
	handler, sender := testHandlerWithMailer()
	register := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"username":"mehmet","email":"mehmet@example.com","password":"password123"}`))
	registerResponse := httptest.NewRecorder()
	handler.ServeHTTP(registerResponse, register)
	if registerResponse.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", registerResponse.Code, registerResponse.Body.String())
	}
	verify := httptest.NewRequest(http.MethodPost, "/verify-email", strings.NewReader(`{"email":"mehmet@example.com","code":"`+sender.verificationCode+`"}`))
	verifyResponse := httptest.NewRecorder()
	handler.ServeHTTP(verifyResponse, verify)
	if verifyResponse.Code != http.StatusOK {
		t.Fatalf("verification failed: %s", verifyResponse.Body.String())
	}
	var sessionCookie *http.Cookie
	for _, cookie := range verifyResponse.Result().Cookies() {
		if cookie.Name == "snipvault_session" {
			sessionCookie = cookie
		}
	}
	if sessionCookie == nil || !sessionCookie.HttpOnly || sessionCookie.SameSite != http.SameSiteStrictMode {
		t.Fatal("expected HttpOnly SameSite=Strict session cookie")
	}

	login := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"mehmet@example.com","password":"password123"}`))
	loginResponse := httptest.NewRecorder()
	handler.ServeHTTP(loginResponse, login)
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", loginResponse.Code, loginResponse.Body.String())
	}
}
