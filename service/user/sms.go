package user

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/user"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/sms"
	"github.com/gin-gonic/gin"
)

// smsEmailDomain is the internal domain used to synthesize a unique email for a
// phone-number account, e.g. "13800000000@sms.local". This lets phone accounts
// reuse the existing unique-email index for O(1) phone -> user lookup without a
// schema change. The domain is never shown to users.
const smsEmailDomain = "sms.local"

// chinaMobilePattern validates a mainland-China mobile number.
var chinaMobilePattern = regexp.MustCompile(`^1[3-9]\d{9}$`)

// smsCodeCacheKey is the KV key holding the pending verification code for a phone.
func smsCodeCacheKey(phone string) string {
	return fmt.Sprintf("sms_code_%s", phone)
}

// smsIntervalCacheKey is the KV key marking that a code was recently sent to a phone.
func smsIntervalCacheKey(phone string) string {
	return fmt.Sprintf("sms_interval_%s", phone)
}

// phoneToEmail maps a phone number to its synthesized internal email.
func phoneToEmail(phone string) string {
	return strings.ToLower(phone) + "@" + smsEmailDomain
}

// normalizePhone trims surrounding whitespace from a submitted phone number.
func normalizePhone(phone string) string {
	return strings.TrimSpace(phone)
}

type (
	// SendSMSCodeService sends a login/registration verification code to a phone.
	SendSMSCodeService struct {
		Phone string `form:"phone" json:"phone" binding:"required"`
	}
	SendSMSCodeParameterCtx struct{}
)

// Send generates a verification code and delivers it via the SMS gateway.
func (service *SendSMSCodeService) Send(c *gin.Context) error {
	dep := dependency.FromContext(c)
	settings := dep.SettingProvider()

	if !settings.SMSLoginEnabled(c) {
		return serializer.NewError(serializer.CodeNoPermissionErr, "Phone login is disabled", nil)
	}

	phone := normalizePhone(service.Phone)
	if !chinaMobilePattern.MatchString(phone) {
		return serializer.NewError(serializer.CodeParamErr, "Invalid phone number", nil)
	}

	cfg := settings.SMS(c)
	kv := dep.KV()

	// Rate limit: refuse if a code was sent within the configured interval.
	if _, ok := kv.Get(smsIntervalCacheKey(phone)); ok {
		return serializer.NewError(serializer.CodeParamErr, "Requesting too frequently, please try again later", nil)
	}

	code, err := generateNumericCode(6)
	if err != nil {
		return serializer.NewError(serializer.CodeInternalSetting, "Failed to generate code", err)
	}

	if err := kv.Set(smsCodeCacheKey(phone), code, cfg.CodeTTL); err != nil {
		return serializer.NewError(serializer.CodeInternalSetting, "Failed to store code", err)
	}
	if err := kv.Set(smsIntervalCacheKey(phone), true, cfg.SendInterval); err != nil {
		return serializer.NewError(serializer.CodeInternalSetting, "Failed to store send marker", err)
	}

	content := strings.ReplaceAll(cfg.Template, "{code}", code)
	if err := sms.Send(c, cfg, phone, content); err != nil {
		// Roll back the interval marker so the user can retry immediately.
		_ = kv.Delete(smsIntervalCacheKey(phone))
		return serializer.NewError(serializer.CodeFailedSendEmail, "Failed to send SMS", err)
	}

	return nil
}

type (
	// SMSLoginService logs in (and auto-registers) a user via phone + code.
	SMSLoginService struct {
		Phone string `form:"phone" json:"phone" binding:"required"`
		Code  string `form:"code" json:"code" binding:"required"`
	}
	SMSLoginParameterCtx struct{}
)

// Login verifies the code and returns the matching user, creating one on first
// login when auto-registration is enabled.
func (service *SMSLoginService) Login(c *gin.Context) (*ent.User, error) {
	dep := dependency.FromContext(c)
	settings := dep.SettingProvider()

	if !settings.SMSLoginEnabled(c) {
		return nil, serializer.NewError(serializer.CodeNoPermissionErr, "Phone login is disabled", nil)
	}

	phone := normalizePhone(service.Phone)
	if !chinaMobilePattern.MatchString(phone) {
		return nil, serializer.NewError(serializer.CodeParamErr, "Invalid phone number", nil)
	}

	kv := dep.KV()
	cached, ok := kv.Get(smsCodeCacheKey(phone))
	if !ok {
		return nil, serializer.NewError(serializer.Code2FACodeErr, "Verification code has expired", nil)
	}
	if expected, _ := cached.(string); expected != strings.TrimSpace(service.Code) {
		return nil, serializer.NewError(serializer.Code2FACodeErr, "Incorrect verification code", nil)
	}

	// Code is single-use.
	_ = kv.Delete(smsCodeCacheKey(phone))
	_ = kv.Delete(smsIntervalCacheKey(phone))

	cfg := settings.SMS(c)
	userClient := dep.UserClient()
	ctx := context.WithValue(c, inventory.LoadUserGroup{}, true)
	email := phoneToEmail(phone)

	expectedUser, err := userClient.GetByEmail(ctx, email)
	if err != nil {
		// User not found: auto-register when enabled.
		if !cfg.AutoRegister {
			return nil, serializer.NewError(serializer.CodeUserNotFound, "This phone number is not registered", err)
		}

		expectedUser, err = service.register(c, dep, phone, email, settings.DefaultGroup(c))
		if err != nil {
			return nil, err
		}

		// Reload with group edges for token issuance / permissions.
		expectedUser, err = userClient.GetByEmail(ctx, email)
		if err != nil {
			return nil, serializer.NewError(serializer.CodeUserNotFound, "User not found", err)
		}
	}

	if expectedUser.Status == user.StatusManualBanned || expectedUser.Status == user.StatusSysBanned {
		return nil, serializer.NewError(serializer.CodeUserBaned, "This account has been blocked", nil)
	}
	if expectedUser.Status == user.StatusInactive {
		return nil, serializer.NewError(serializer.CodeUserNotActivated, "This account is not activated", nil)
	}

	return expectedUser, nil
}

// register creates a new phone-number account inside a transaction.
func (service *SMSLoginService) register(c *gin.Context, dep dependency.Dep, phone, email string, groupID int) (*ent.User, error) {
	args := &inventory.NewUserArgs{
		Email:   email,
		Nick:    phone,
		Status:  user.StatusActive,
		GroupID: groupID,
		Phone:   phone,
	}

	uc, tx, _, err := inventory.WithTx(c, dep.UserClient())
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to start transaction", err)
	}

	newUser, err := uc.Create(c, args)
	if err != nil {
		_ = inventory.Rollback(tx)
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to create user", err)
	}

	if err := inventory.Commit(tx); err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to commit user", err)
	}

	return newUser, nil
}

// generateNumericCode returns a cryptographically-random decimal string of n digits.
func generateNumericCode(n int) (string, error) {
	var sb strings.Builder
	sb.Grow(n)
	for i := 0; i < n; i++ {
		d, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		sb.WriteByte(byte('0' + d.Int64()))
	}
	return sb.String(), nil
}
