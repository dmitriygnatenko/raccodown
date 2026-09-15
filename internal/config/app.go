package config

// Application defaults, used whenever the corresponding env var isn't set. The demo credentials
// only ever reach a freshly seeded account (see app.seedDemoUser), never an existing one.
const (
	defaultPort         = "8080"
	defaultDemoUsername = "user"
	defaultDemoPassword = "pass"
)

// AppConfig collects how the server itself runs: where to listen, whether the session cookie is
// HTTPS-only, and the credentials of the demo account seeded on first run. Logging lives in
// LogConfig, loaded separately.
type AppConfig struct {
	Port         string
	CookieSecure bool
	DemoUsername string
	DemoPassword string
}

// LoadApp builds the server configuration from the environment, rejecting anything malformed as it
// goes. Every setting here is optional and falls back to a default — the app is meant to run with
// an empty environment — but a value that is set and malformed (PORT=http, COOKIE_SECURE=yes) is an
// error rather than a silent fallback.
func LoadApp() (AppConfig, error) {
	port, err := portEnv("PORT", defaultPort)
	if err != nil {
		return AppConfig{}, err
	}

	cookieSecure, err := boolEnv("COOKIE_SECURE", false)
	if err != nil {
		return AppConfig{}, err
	}

	return AppConfig{
		Port:         port,
		CookieSecure: cookieSecure,
		DemoUsername: stringEnv("DEMO_USERNAME", defaultDemoUsername),
		DemoPassword: stringEnv("DEMO_PASSWORD", defaultDemoPassword),
	}, nil
}
