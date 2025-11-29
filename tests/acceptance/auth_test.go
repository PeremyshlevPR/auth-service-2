package acceptance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prperemyshlev/auth-service-2/internal/dto"
)

func (s *Suite) TestRegister_Success() {
	reqBody := dto.RegisterRequest{
		Email:    "test@example.com",
		Password: "Password123",
	}
	body, _ := json.Marshal(reqBody)

	resp, err := http.Post(
		s.BaseURL+"/api/v1/auth/register",
		"application/json",
		bytes.NewBuffer(body),
	)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(http.StatusCreated, resp.StatusCode)

	var authResp dto.AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&authResp)
	s.Require().NoError(err)

	s.NotEmpty(authResp.AccessToken)
	s.Equal("Bearer", authResp.TokenType)
	s.NotZero(authResp.ExpiresIn)
	s.Equal("test@example.com", authResp.User.Email)
	s.NotEmpty(authResp.User.ID)

	cookies := resp.Cookies()
	s.NotEmpty(cookies, "Should have refresh token cookie")
}

func (s *Suite) TestRegister_DuplicateEmail() {
	reqBody := dto.RegisterRequest{
		Email:    "duplicate@example.com",
		Password: "Password123",
	}
	body, _ := json.Marshal(reqBody)

	resp1, _ := http.Post(
		s.BaseURL+"/api/v1/auth/register",
		"application/json",
		bytes.NewBuffer(body),
	)
	resp1.Body.Close()

	body, _ = json.Marshal(reqBody)
	resp2, err := http.Post(
		s.BaseURL+"/api/v1/auth/register",
		"application/json",
		bytes.NewBuffer(body),
	)
	s.Require().NoError(err)
	defer resp2.Body.Close()

	s.Equal(http.StatusConflict, resp2.StatusCode)

	var errResp dto.ErrorResponse
	json.NewDecoder(resp2.Body).Decode(&errResp)
	s.Equal("Conflict", errResp.Error)
}

func (s *Suite) TestRegister_InvalidEmail() {
	reqBody := dto.RegisterRequest{
		Email:    "invalid-email",
		Password: "Password123",
	}
	body, _ := json.Marshal(reqBody)

	resp, err := http.Post(
		s.BaseURL+"/api/v1/auth/register",
		"application/json",
		bytes.NewBuffer(body),
	)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (s *Suite) TestRegister_ShortPassword() {
	reqBody := dto.RegisterRequest{
		Email:    "test@example.com",
		Password: "short",
	}
	body, _ := json.Marshal(reqBody)

	resp, err := http.Post(
		s.BaseURL+"/api/v1/auth/register",
		"application/json",
		bytes.NewBuffer(body),
	)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (s *Suite) TestLogin_Success() {
	registerReq := dto.RegisterRequest{
		Email:    "login@example.com",
		Password: "Password123",
	}
	body, _ := json.Marshal(registerReq)
	registerResp, err := http.Post(s.BaseURL+"/api/v1/auth/register", "application/json", bytes.NewBuffer(body))
	s.Require().NoError(err)
	defer registerResp.Body.Close()
	s.Equal(http.StatusCreated, registerResp.StatusCode, "Registration should succeed")

	loginReq := dto.LoginRequest{
		Email:    "login@example.com",
		Password: "Password123",
	}
	body, _ = json.Marshal(loginReq)

	resp, err := http.Post(
		s.BaseURL+"/api/v1/auth/login",
		"application/json",
		bytes.NewBuffer(body),
	)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(http.StatusOK, resp.StatusCode)

	var authResp dto.AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&authResp)
	s.Require().NoError(err)

	s.NotEmpty(authResp.AccessToken)
	s.Equal("Bearer", authResp.TokenType)
	s.Equal("login@example.com", authResp.User.Email)

	cookies := resp.Cookies()
	s.NotEmpty(cookies, "Should have refresh token cookie")
}

func (s *Suite) TestLogin_InvalidCredentials() {
	loginReq := dto.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "wrongpassword",
	}
	body, _ := json.Marshal(loginReq)

	resp, err := http.Post(
		s.BaseURL+"/api/v1/auth/login",
		"application/json",
		bytes.NewBuffer(body),
	)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(http.StatusUnauthorized, resp.StatusCode)

	var errResp dto.ErrorResponse
	json.NewDecoder(resp.Body).Decode(&errResp)
	s.Equal("Unauthorized", errResp.Error)
}

func (s *Suite) TestLogin_WrongPassword() {
	registerReq := dto.RegisterRequest{
		Email:    "wrongpass@example.com",
		Password: "CorrectPassword123",
	}
	body, _ := json.Marshal(registerReq)
	http.Post(s.BaseURL+"/api/v1/auth/register", "application/json", bytes.NewBuffer(body))

	loginReq := dto.LoginRequest{
		Email:    "wrongpass@example.com",
		Password: "WrongPassword123",
	}
	body, _ = json.Marshal(loginReq)

	resp, err := http.Post(
		s.BaseURL+"/api/v1/auth/login",
		"application/json",
		bytes.NewBuffer(body),
	)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *Suite) TestGetMe_Success() {
	registerReq := dto.RegisterRequest{
		Email:    "getme@example.com",
		Password: "Password123",
	}
	body, _ := json.Marshal(registerReq)
	registerResp, err := http.Post(
		s.BaseURL+"/api/v1/auth/register",
		"application/json",
		bytes.NewBuffer(body),
	)
	s.Require().NoError(err)
	defer registerResp.Body.Close()

	var authResp dto.AuthResponse
	json.NewDecoder(registerResp.Body).Decode(&authResp)

	req, _ := http.NewRequest("GET", s.BaseURL+"/api/v1/auth/me", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authResp.AccessToken))

	resp, err := http.DefaultClient.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(http.StatusOK, resp.StatusCode)

	var userResp dto.UserResponse
	err = json.NewDecoder(resp.Body).Decode(&userResp)
	s.Require().NoError(err)

	s.NotEmpty(userResp.ID)
	s.Equal("getme@example.com", userResp.Email)
	s.NotEmpty(userResp.CreatedAt)
	s.NotEmpty(userResp.UpdatedAt)
	s.False(userResp.IsEmailVerified)
}

func (s *Suite) TestGetMe_NoToken() {
	req, _ := http.NewRequest("GET", s.BaseURL+"/api/v1/auth/me", nil)

	resp, err := http.DefaultClient.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *Suite) TestGetMe_InvalidToken() {
	req, _ := http.NewRequest("GET", s.BaseURL+"/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")

	resp, err := http.DefaultClient.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *Suite) TestLogout_Success() {
	registerReq := dto.RegisterRequest{
		Email:    "logout@example.com",
		Password: "Password123",
	}
	body, _ := json.Marshal(registerReq)
	registerResp, err := http.Post(
		s.BaseURL+"/api/v1/auth/register",
		"application/json",
		bytes.NewBuffer(body),
	)
	s.Require().NoError(err)
	defer registerResp.Body.Close()

	var authResp dto.AuthResponse
	json.NewDecoder(registerResp.Body).Decode(&authResp)

	req, _ := http.NewRequest("POST", s.BaseURL+"/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authResp.AccessToken))

	resp, err := http.DefaultClient.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(http.StatusOK, resp.StatusCode)

	var successResp dto.SuccessResponse
	json.NewDecoder(resp.Body).Decode(&successResp)
	s.Equal("Logged out successfully", successResp.Message)
}

func (s *Suite) TestLogout_NoToken() {
	req, _ := http.NewRequest("POST", s.BaseURL+"/api/v1/auth/logout", nil)

	resp, err := http.DefaultClient.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *Suite) TestRefresh_Success() {
	registerReq := dto.RegisterRequest{
		Email:    "refresh@example.com",
		Password: "Password123",
	}
	body, _ := json.Marshal(registerReq)
	registerResp, err := http.Post(
		s.BaseURL+"/api/v1/auth/register",
		"application/json",
		bytes.NewBuffer(body),
	)
	s.Require().NoError(err)
	defer registerResp.Body.Close()

	cookies := registerResp.Cookies()
	s.Require().NotEmpty(cookies)

	req, _ := http.NewRequest("POST", s.BaseURL+"/api/v1/auth/refresh", nil)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp, err := http.DefaultClient.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(http.StatusOK, resp.StatusCode)

	var authResp dto.AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&authResp)
	s.Require().NoError(err)

	s.NotEmpty(authResp.AccessToken)
	s.Equal("Bearer", authResp.TokenType)
}

func (s *Suite) TestRefresh_NoCookie() {
	req, _ := http.NewRequest("POST", s.BaseURL+"/api/v1/auth/refresh", nil)

	resp, err := http.DefaultClient.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (s *Suite) TestCompleteFlow() {
	email := "complete@example.com"
	password := "Password123"

	registerReq := dto.RegisterRequest{
		Email:    email,
		Password: password,
	}
	body, _ := json.Marshal(registerReq)
	registerResp, err := http.Post(
		s.BaseURL+"/api/v1/auth/register",
		"application/json",
		bytes.NewBuffer(body),
	)
	s.Require().NoError(err)
	defer registerResp.Body.Close()
	s.Equal(http.StatusCreated, registerResp.StatusCode)

	var authResp dto.AuthResponse
	json.NewDecoder(registerResp.Body).Decode(&authResp)
	accessToken := authResp.AccessToken

	req, _ := http.NewRequest("GET", s.BaseURL+"/api/v1/auth/me", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	meResp, err := http.DefaultClient.Do(req)
	s.Require().NoError(err)
	defer meResp.Body.Close()
	s.Equal(http.StatusOK, meResp.StatusCode)

	cookies := registerResp.Cookies()
	refreshReq, _ := http.NewRequest("POST", s.BaseURL+"/api/v1/auth/refresh", nil)
	for _, cookie := range cookies {
		refreshReq.AddCookie(cookie)
	}
	refreshResp, err := http.DefaultClient.Do(refreshReq)
	s.Require().NoError(err)
	defer refreshResp.Body.Close()
	s.Equal(http.StatusOK, refreshResp.StatusCode)

	var newAuthResp dto.AuthResponse
	json.NewDecoder(refreshResp.Body).Decode(&newAuthResp)
	newAccessToken := newAuthResp.AccessToken

	logoutReq, _ := http.NewRequest("POST", s.BaseURL+"/api/v1/auth/logout", nil)
	logoutReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", newAccessToken))
	logoutResp, err := http.DefaultClient.Do(logoutReq)
	s.Require().NoError(err)
	defer logoutResp.Body.Close()
	s.Equal(http.StatusOK, logoutResp.StatusCode)

	meReq2, _ := http.NewRequest("GET", s.BaseURL+"/api/v1/auth/me", nil)
	meReq2.Header.Set("Authorization", fmt.Sprintf("Bearer %s", newAccessToken))
	meResp2, err := http.DefaultClient.Do(meReq2)
	s.Require().NoError(err)
	defer meResp2.Body.Close()
	s.Equal(http.StatusOK, meResp2.StatusCode)
}
