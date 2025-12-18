package config

// Config holds basic configuration such as database DSN.
type Config struct {
	DSN string
}

// Load loads configuration. For this demo we hardcode values.
// You can switch to env vars or config files as needed.
func Load() *Config {
	return &Config{
		// Adjust DSN to match your local MySQL settings.
		// Format: username:password@tcp(host:port)/dbname?parseTime=true&loc=Local
		DSN: "root:root@tcp(127.0.0.1:3306)/appstats?parseTime=true&loc=Local",
	}
}
