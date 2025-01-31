package config

import (
	"crypto/rsa"
	"fmt"
	"github.com/eurofurence/reg-attendee-service/internal/web/util/validation"
	"github.com/golang-jwt/jwt/v4"
	"net/url"
	"strings"
	"time"
)

func setConfigurationDefaults(c *conf) {
	if c.Server.Port == "" {
		c.Server.Port = "8080"
	}
	if c.Server.ReadTimeout <= 0 {
		c.Server.ReadTimeout = 5
	}
	if c.Server.WriteTimeout <= 0 {
		c.Server.WriteTimeout = 5
	}
	if c.Server.IdleTimeout <= 0 {
		c.Server.IdleTimeout = 5
	}
	if c.Logging.Severity == "" {
		c.Logging.Severity = "INFO"
	}
	if c.Database.Use == "" {
		c.Database.Use = "inmemory"
	}
	if c.Security.CorsAllowOrigin == "" {
		c.Security.CorsAllowOrigin = "*"
	}
}

const portPattern = "^[1-9][0-9]{0,4}$"

func validateServerConfiguration(errs url.Values, c serverConfig) {
	if validation.ViolatesPattern(portPattern, c.Port) {
		errs.Add("server.port", "must be a number between 1 and 65535")
	}
	validation.CheckIntValueRange(&errs, 1, 300, "server.read_timeout_seconds", c.ReadTimeout)
	validation.CheckIntValueRange(&errs, 1, 300, "server.write_timeout_seconds", c.WriteTimeout)
	validation.CheckIntValueRange(&errs, 1, 300, "server.idle_timeout_seconds", c.IdleTimeout)
}

var allowedSeverities = []string{"DEBUG", "INFO", "WARN", "ERROR"}

func validateLoggingConfiguration(errs url.Values, c loggingConfig) {
	if validation.NotInAllowedValues(allowedSeverities[:], c.Severity) {
		errs.Add("logging.severity", "must be one of DEBUG, INFO, WARN, ERROR")
	}
}

func validateSecurityConfiguration(errs url.Values, c securityConfig) {
	validation.CheckLength(&errs, 16, 256, "security.fixed.api", c.Fixed.Api)
	validation.CheckLength(&errs, 1, 256, "security.oidc.admin_role", c.Oidc.AdminRole)

	parsedKeySet = make([]*rsa.PublicKey, 0)
	for i, keyStr := range c.Oidc.TokenPublicKeysPEM {
		publicKeyPtr, err := jwt.ParseRSAPublicKeyFromPEM([]byte(keyStr))
		if err != nil {
			errs.Add(fmt.Sprintf("security.oidc.token_public_keys_PEM[%d]", i), fmt.Sprintf("failed to parse RSA public key in PEM format: %s", err.Error()))
		} else {
			parsedKeySet = append(parsedKeySet, publicKeyPtr)
		}
	}
}

var allowedDatabases = []string{"mysql", "inmemory"}

func validateDatabaseConfiguration(errs url.Values, c databaseConfig) {
	if validation.NotInAllowedValues(allowedDatabases[:], c.Use) {
		errs.Add("database.use", "must be one of mysql, inmemory")
	}
	if c.Use == "mysql" {
		validation.CheckLength(&errs, 1, 256, "database.mysql.username", c.Mysql.Username)
		validation.CheckLength(&errs, 1, 256, "database.mysql.password", c.Mysql.Password)
		validation.CheckLength(&errs, 1, 256, "database.mysql.database", c.Mysql.Database)
	}
}

func validateBirthdayConfiguration(errs url.Values, c birthdayConfig) {
	if validation.InvalidISODate(c.Earliest) {
		errs.Add("birthday.earliest", "invalid earliest birthday, must be specified as an ISO Date, as in 1901-01-01")
	}
	if validation.InvalidISODate(c.Latest) {
		errs.Add("birthday.latest", "invalid latest birthday, must be specified as an ISO Date, as in 2019-08-24. It is acceptable to specify the last day of the convention, if you wish to allow any underage participants. Otherwise use the first day, 18 years ago.")
	}
}

const keyPattern = "^[a-zA-Z0-9_-]+$"

func validateFlagsConfiguration(errs url.Values, c map[string]ChoiceConfig) {
	for k, v := range c {
		if validation.ViolatesPattern(keyPattern, k) {
			errs.Add("choices.flags."+k, "invalid key, must consist of a-z A-Z 0-9 - _ only")
		}
		validation.CheckLength(&errs, 1, 256, "choices.flags."+k+".description", v.Description)
		validation.CheckLength(&errs, 1, 256, "choices.flags."+k+".help_url", v.HelpUrl)
		checkConstraints(errs, c, "choices.flags", k, v.Constraint, v.ConstraintMsg)
		if v.AdminOnly && v.ReadOnly {
			errs.Add("choices.flags."+k+".admin", "a flag cannot both be admin_only and read_only")
		}
		if v.AdminOnly && v.Default {
			errs.Add("choices.flags."+k+".default", "a flag cannot both be admin_only and default to on")
		}
	}
}

func validatePackagesConfiguration(errs url.Values, c map[string]ChoiceConfig) {
	for k, v := range c {
		if validation.ViolatesPattern(keyPattern, k) {
			errs.Add("choices.packages."+k, "invalid key, must consist of a-z A-Z 0-9 - _ only")
		}
		validation.CheckLength(&errs, 1, 256, "choices.packages."+k+".description", v.Description)
		validation.CheckLength(&errs, 1, 256, "choices.packages."+k+".help_url", v.HelpUrl)
		checkConstraints(errs, c, "choices.packages", k, v.Constraint, v.ConstraintMsg)
		if v.AdminOnly {
			errs.Add("choices.packages."+k+".admin", "packages cannot be admin_only (they cost money). Try read_only instead.")
		}
	}
}

func validateOptionsConfiguration(errs url.Values, c map[string]ChoiceConfig) {
	for k, v := range c {
		if validation.ViolatesPattern(keyPattern, k) {
			errs.Add("choices.options."+k, "invalid key, must consist of a-z A-Z 0-9 - _ only")
		}
		validation.CheckLength(&errs, 1, 256, "choices.options."+k+".description", v.Description)
		validation.CheckLength(&errs, 1, 256, "choices.options."+k+".help_url", v.HelpUrl)
		checkConstraints(errs, c, "choices.options", k, v.Constraint, v.ConstraintMsg)
		if v.AdminOnly {
			errs.Add("choices.options."+k+".admin", "options cannot be admin_only (they represent user choices).")
		}
		if v.ReadOnly {
			errs.Add("choices.options."+k+".readonly", "options cannot be read_only (they represent user choices).")
		}
	}
}

func checkConstraints(errs url.Values, c map[string]ChoiceConfig, keyPrefix string, key string, constraint string, constraintMsg string) {
	if constraint != "" {
		constraints := strings.Split(constraint, ",")
		for _, cn := range constraints {
			choiceKey := cn
			if strings.HasPrefix(cn, "!") {
				choiceKey = strings.TrimPrefix(cn, "!")
			}
			if _, ok := c[choiceKey]; !ok {
				errs.Add(keyPrefix+"."+key+".constraint", "invalid key in constraint, references nonexistent entry")
			} else {
				if c[choiceKey].AdminOnly != c[key].AdminOnly {
					errs.Add(keyPrefix+"."+key+".constraint", "invalid key in constraint, references across admin only and non-admin only")
				}
			}
			if choiceKey == key {
				errs.Add(keyPrefix+"."+key+".constraint", "invalid self referential constraint")
			}
			validation.CheckLength(&errs, 1, 256, keyPrefix+"."+key+".constraint_msg", constraintMsg)
		}
	}
}

func validateRegistrationStartTime(errs url.Values, c goLiveConfig, s securityConfig) {
	normal, err := time.Parse(StartTimeFormat, c.StartIsoDatetime)
	if err != nil {
		errs.Add("go_live.start_iso_datetime", "invalid date/time format, use ISO with numeric timezone as in "+StartTimeFormat)
	}

	if c.EarlyRegStartIsoDatetime != "" {
		early, err := time.Parse(StartTimeFormat, c.EarlyRegStartIsoDatetime)
		if err != nil {
			errs.Add("go_live.early_reg_start_iso_datetime", "invalid date/time format, use ISO with numeric timezone as in "+StartTimeFormat)
		}

		if normal.Before(early) {
			errs.Add("go_live.early_reg_start_iso_datetime", "if supplied, must be earlier than go_live.start_iso_datetime")
		}

		if s.Oidc.EarlyReg == "" {
			errs.Add("go_live.early_reg_start_iso_datetime", "if supplied, must also supply security.oidc.early_reg_role so early registration is possible")
		}
	}
}

const downstreamPattern = "^(|https?://.*[^/])$"

func validateDownstreamConfiguration(errs url.Values, c downstreamConfig) {
	if validation.ViolatesPattern(downstreamPattern, c.PaymentService) {
		errs.Add("downstream.payment_service", "base url must be empty (enables in-memory simulator) or start with http:// or https:// and may not end in a /")
	}
	if validation.ViolatesPattern(downstreamPattern, c.MailService) {
		errs.Add("downstream.payment_service", "base url must be empty (enables in-memory simulator) or start with http:// or https:// and may not end in a /")
	}
}
