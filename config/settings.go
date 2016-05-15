package config

import (
	"github.com/getblank/blank-sr/bdb"
	"runtime"
)

var (
	CommonSettings *CommonSettingsStructure
	ServerSettings *ServerSettingsStructure
	ClientSettings bdb.M
)

type CommonSettingsStructure struct {
	UserActivation bool                   `json:"userActivation,omitempty"`
	BaseUrl        string                 `json:"baseUrl,omitempty"`
	DefaultLocale  string                 `json:"defaultLocale,omitempty"`
	I18n           map[string]interface{} `json:"i18n,omitempty"`
	LessVars       map[string]interface{} `json:"lessVars,omitempty"`
}

type ServerSettingsStructure struct {
	RegisterTokenExpiration           int     `json:"registerTokenExpiration,omitempty"`
	PasswordResetTokenExpiration      int     `json:"passwordResetTokenExpiration,omitempty"`
	ActivationEmailTemplate           string  `json:"activationEmailTemplate,omitempty"`
	PasswordResetEmailTemplate        string  `json:"passwordResetEmailTemplate,omitempty"`
	PasswordResetSuccessEmailTemplate string  `json:"passwordResetSuccessEmailTemplate,omitempty"`
	RegistrationSuccessEmailTemplate  string  `json:"registrationSuccessEmailTemplate,omitempty"`
	ActivationSuccessPage             string  `json:"activationSuccessPage,omitempty"`
	ActivationErrorPage               string  `json:"activationErrorPage,omitempty"`
	FileStorePath                     string  `json:"fileStorePath,omitempty"`
	VMPoolSize                        int     `json:"vmPoolSize,omitempty"`
	MaxLogSize                        int     `json:"maxLogSize,omitempty"`
	MemoryCheckingInterval            int     `json:"memoryCheckingInterval,omitempty"`
	MemoryUsageToCleanWorkers         float64 `json:"memoryUsageToCleanWorkers,omitempty"`
	Port                              string  `json:"port,omitempty"`
	DBDriver						  string  `json:"dbDriver,omitempty"`
	MongoURI                          string  `json:"mongoURI,omitempty"`
}

func makeDefaultSettings() {
	CommonSettings = &CommonSettingsStructure{
		BaseUrl:       "http://localhost:3001",
		DefaultLocale: "en",
		LessVars:      map[string]interface{}{},
	}
	ServerSettings = &ServerSettingsStructure{
		RegisterTokenExpiration:           60,
		PasswordResetTokenExpiration:      60,
		ActivationEmailTemplate:           "./templates/activation-email.html",
		ActivationSuccessPage:             "./templates/activation-success.html",
		ActivationErrorPage:               "./templates/activation-error.html",
		PasswordResetEmailTemplate:        "./templates/password-reset-email.html",
		PasswordResetSuccessEmailTemplate: "/templates/password-reset-success-email.html",
		RegistrationSuccessEmailTemplate:  "./templates/registration-success-email.html",
		FileStorePath:                     "./files",
		VMPoolSize:                        runtime.NumCPU() * 4,
		MaxLogSize:                        1000,
		MemoryCheckingInterval:            10,
		MemoryUsageToCleanWorkers:         80,
		Port:     "3001",
		DBDriver: "bolt",
		MongoURI: "mongodb://localhost/blank",
	}
}
