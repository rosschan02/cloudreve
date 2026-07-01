// Package sms sends verification-code text messages through the 联通 (028lk)
// SMS gateway. See docs: https://api.028lk.com/Sms/Api/Send
package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
)

// sendRequest is the JSON body of the 028lk Send API (明文鉴权：SecretName+SecretKey，
// TimeStamp 无需填写).
type sendRequest struct {
	SecretName string `json:"SecretName"`
	SecretKey  string `json:"SecretKey"`
	Mobile     string `json:"Mobile"`
	Content    string `json:"Content"`
	SignName   string `json:"SignName,omitempty"`
}

// sendResponse is the JSON response of the 028lk Send API. code == 0 means success.
type sendResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data string `json:"data"`
}

// Send delivers content to a single mobile number through the gateway described
// by cfg. It returns an error when the gateway is not configured or rejects the
// request.
func Send(ctx context.Context, cfg *setting.SMS, mobile, content string) error {
	if cfg == nil || cfg.SecretName == "" || cfg.SecretKey == "" {
		return fmt.Errorf("sms gateway is not configured")
	}

	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "https://api.028lk.com/Sms/Api/Send"
	}

	payload, err := json.Marshal(&sendRequest{
		SecretName: cfg.SecretName,
		SecretKey:  cfg.SecretKey,
		Mobile:     mobile,
		Content:    content,
		SignName:   cfg.SignName,
	})
	if err != nil {
		return fmt.Errorf("failed to encode sms request: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to build sms request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach sms gateway: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("sms gateway returned HTTP %d", resp.StatusCode)
	}

	var parsed sendResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return fmt.Errorf("failed to decode sms response: %w", err)
	}

	if parsed.Code != 0 {
		return fmt.Errorf("sms gateway error %d: %s", parsed.Code, parsed.Msg)
	}

	return nil
}
