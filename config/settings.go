package config

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/getblank/blank-sr/bdb"
)

var (
	commonSettings *commonSettingsStruct
	serverSettings *serverSettingsStruct
	clientSettings bdb.M

	// ErrInvalidTTLFormat represents ttl format error
	ErrInvalidTTLFormat = errors.New("invalid ttl in config")
)

// JWTTTL returns TTL for JWT tokens
func JWTTTL() (time.Duration, error) {
	confLocker.RLock()
	defer confLocker.RUnlock()
	// For testing purpose
	if serverSettings == nil {
		return time.Hour * 24, nil
	}
	if serverSettings.jwtTTL != nil {
		return *serverSettings.jwtTTL, nil
	}
	ttlStrings := strings.Split(serverSettings.JWTTTL, ":")
	if len(ttlStrings) == 0 {
		return 0, ErrInvalidTTLFormat
	}
	hours, err := strconv.Atoi(ttlStrings[0])
	if err != nil {
		return 0, err
	}
	res := time.Hour * time.Duration(hours)
	var minutes int
	if len(ttlStrings) > 1 {
		minutes, err = strconv.Atoi(ttlStrings[1])
		if err != nil {
			return 0, err
		}
		res = res + time.Minute*time.Duration(minutes)
	}
	serverSettings.jwtTTL = &res
	return res, nil
}

type commonSettingsStruct struct {
	UserActivation bool                   `json:"userActivation,omitempty"`
	BaseURL        string                 `json:"baseUrl,omitempty"`
	DefaultLocale  string                 `json:"defaultLocale,omitempty"`
	I18n           map[string]interface{} `json:"i18n,omitempty"`
	LessVars       map[string]interface{} `json:"lessVars,omitempty"`
}

type serverSettingsStruct struct {
	RegisterTokenExpiration           string   `json:"registerTokenExpiration,omitempty"`
	PasswordResetTokenExpiration      string   `json:"passwordResetTokenExpiration,omitempty"`
	ActivationEmailTemplate           string   `json:"activationEmailTemplate,omitempty"`
	PasswordResetEmailTemplate        string   `json:"passwordResetEmailTemplate,omitempty"`
	PasswordResetSuccessEmailTemplate string   `json:"passwordResetSuccessEmailTemplate,omitempty"`
	RegistrationSuccessEmailTemplate  string   `json:"registrationSuccessEmailTemplate,omitempty"`
	ActivationSuccessPage             string   `json:"activationSuccessPage,omitempty"`
	ActivationErrorPage               string   `json:"activationErrorPage,omitempty"`
	MaxLogSize                        int      `json:"maxLogSize,omitempty"`
	Port                              string   `json:"port,omitempty"`
	SSOOrigins                        []string `json:"ssoOrigins,omitempty"`
	JWTTTL                            string   `json:"jwtTtl,omitempty"`
	jwtTTL                            *time.Duration
}

func makeDefaultSettings() {
	commonSettings = &commonSettingsStruct{
		BaseURL:       "http://localhost:8080",
		DefaultLocale: "en",
		LessVars:      map[string]interface{}{},
	}
	serverSettings = &serverSettingsStruct{
		RegisterTokenExpiration:           "0:60",
		PasswordResetTokenExpiration:      "0:60",
		ActivationEmailTemplate:           "./templates/activation-email.html",
		ActivationSuccessPage:             "./templates/activation-success.html",
		ActivationErrorPage:               "./templates/activation-error.html",
		PasswordResetEmailTemplate:        "./templates/password-reset-email.html",
		PasswordResetSuccessEmailTemplate: "/templates/password-reset-success-email.html",
		RegistrationSuccessEmailTemplate:  "./templates/registration-success-email.html",
		MaxLogSize:                        1000,
		Port:                              "3001",
		JWTTTL:                            "24:00",
	}
}
