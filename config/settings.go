package config

import "github.com/getblank/blank-sr/bdb"

var (
	CommonSettings *CommonSettingsStructure
	ServerSettings *ServerSettingsStructure
	ClientSettings bdb.M
)

type CommonSettingsStructure struct {
	UserActivation bool                   `json:"userActivation,omitempty"`
	BaseURL        string                 `json:"baseUrl,omitempty"`
	DefaultLocale  string                 `json:"defaultLocale,omitempty"`
	I18n           map[string]interface{} `json:"i18n,omitempty"`
	LessVars       map[string]interface{} `json:"lessVars,omitempty"`
}

type ServerSettingsStructure struct {
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
}

func makeDefaultSettings() {
	CommonSettings = &CommonSettingsStructure{
		BaseURL:       "http://localhost:8080",
		DefaultLocale: "en",
		LessVars:      map[string]interface{}{},
	}
	ServerSettings = &ServerSettingsStructure{
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
	}
}
