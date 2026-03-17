package policy

// DefaultNeverShare returns patterns that should never be shared.
func DefaultNeverShare() []string {
	return []string{
		// Credentials and secrets
		".env",
		".env.*",
		"*.pem",
		"*.key",
		"*.p12",
		"*.pfx",
		"credentials.json",
		"service-account.json",
		"id_rsa",
		"id_ed25519",
		".netrc",
		".npmrc",
		".pypirc",

		// SSH and GPG
		".ssh",
		".gnupg",

		// Git internals
		".git",

		// OS and IDE
		".DS_Store",
		"Thumbs.db",
		".idea",
		".vscode",

		// Package manager caches
		"node_modules",
		".cache",
		"__pycache__",
		".tox",
		".venv",
		"venv",

		// Build outputs
		"dist",
		"build",
		"target",

		// Databases
		"*.sqlite",
		"*.sqlite3",
		"*.db",

		// Keychains
		"*.keychain",
		"*.keychain-db",
		"login.keychain",

		// Browser data
		"Cookies",
		"Login Data",
		"Web Data",
	}
}
