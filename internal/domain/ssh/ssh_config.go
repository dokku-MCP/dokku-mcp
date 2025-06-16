package ssh

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// SSHConfig représente une configuration SSH validée et sécurisée pour Dokku
type SSHConfig struct {
	host     string
	port     int
	user     string
	keyPath  string
	timeout  time.Duration
	verified bool
}

var (
	// Pattern pour valider les noms d'hôte
	hostnamePattern = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9.-]{0,61}[a-zA-Z0-9])?$`)
	// Pattern pour valider les noms d'utilisateur SSH
	usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
)

// NewSSHConfig crée une nouvelle configuration SSH avec validation
func NewSSHConfig(host string, port int, user string, keyPath string, timeout time.Duration) (*SSHConfig, error) {
	host = strings.TrimSpace(host)
	user = strings.TrimSpace(user)
	keyPath = strings.TrimSpace(keyPath)

	if err := validateSSHConfig(host, port, user, keyPath, timeout); err != nil {
		return nil, fmt.Errorf("configuration SSH invalide: %w", err)
	}

	return &SSHConfig{
		host:     host,
		port:     port,
		user:     user,
		keyPath:  keyPath,
		timeout:  timeout,
		verified: false,
	}, nil
}

// NewDefaultSSHConfig crée une configuration SSH par défaut pour Dokku
func NewDefaultSSHConfig() *SSHConfig {
	return &SSHConfig{
		host:    "dokku.me",
		port:    22,
		user:    "dokku",
		keyPath: "", // Utilise l'agent SSH ou la clé par défaut
		timeout: 30 * time.Second,
	}
}

// MustNewSSHConfig crée une configuration SSH en paniquant en cas d'erreur
func MustNewSSHConfig(host string, port int, user string, keyPath string, timeout time.Duration) *SSHConfig {
	config, err := NewSSHConfig(host, port, user, keyPath, timeout)
	if err != nil {
		panic(fmt.Sprintf("impossible de créer la configuration SSH: %v", err))
	}
	return config
}

// Host retourne l'hôte SSH
func (s *SSHConfig) Host() string {
	return s.host
}

// Port retourne le port SSH
func (s *SSHConfig) Port() int {
	return s.port
}

// User retourne l'utilisateur SSH
func (s *SSHConfig) User() string {
	return s.user
}

// KeyPath retourne le chemin vers la clé SSH
func (s *SSHConfig) KeyPath() string {
	return s.keyPath
}

// Timeout retourne le timeout de connexion
func (s *SSHConfig) Timeout() time.Duration {
	return s.timeout
}

// IsVerified retourne si la configuration a été vérifiée
func (s *SSHConfig) IsVerified() bool {
	return s.verified
}

// ConnectionString retourne la chaîne de connexion SSH
func (s *SSHConfig) ConnectionString() string {
	return fmt.Sprintf("%s@%s:%d", s.user, s.host, s.port)
}

// SSHCommand retourne la commande SSH de base
func (s *SSHConfig) SSHCommand(command string) []string {
	args := []string{
		"ssh",
		"-o", "LogLevel=QUIET",
		"-o", "StrictHostKeyChecking=no",
		"-o", fmt.Sprintf("ConnectTimeout=%d", int(s.timeout.Seconds())),
		"-p", fmt.Sprintf("%d", s.port),
	}

	// Ajouter la clé SSH si spécifiée
	if s.keyPath != "" {
		args = append(args, "-i", s.keyPath)
	}

	// Ajouter la destination
	args = append(args, s.ConnectionString())

	// Ajouter la commande
	if command != "" {
		args = append(args, "--", command)
	}

	return args
}

// WithHost retourne une nouvelle configuration avec un hôte différent
func (s *SSHConfig) WithHost(host string) (*SSHConfig, error) {
	return NewSSHConfig(host, s.port, s.user, s.keyPath, s.timeout)
}

// WithPort retourne une nouvelle configuration avec un port différent
func (s *SSHConfig) WithPort(port int) (*SSHConfig, error) {
	return NewSSHConfig(s.host, port, s.user, s.keyPath, s.timeout)
}

// WithUser retourne une nouvelle configuration avec un utilisateur différent
func (s *SSHConfig) WithUser(user string) (*SSHConfig, error) {
	return NewSSHConfig(s.host, s.port, user, s.keyPath, s.timeout)
}

// WithKeyPath retourne une nouvelle configuration avec un chemin de clé différent
func (s *SSHConfig) WithKeyPath(keyPath string) (*SSHConfig, error) {
	return NewSSHConfig(s.host, s.port, s.user, keyPath, s.timeout)
}

// WithTimeout retourne une nouvelle configuration avec un timeout différent
func (s *SSHConfig) WithTimeout(timeout time.Duration) (*SSHConfig, error) {
	return NewSSHConfig(s.host, s.port, s.user, s.keyPath, timeout)
}

// IsLocalhost vérifie si l'hôte est localhost
func (s *SSHConfig) IsLocalhost() bool {
	return s.host == "localhost" || s.host == "127.0.0.1" || s.host == "::1"
}

// UsesDefaultKey vérifie si la configuration utilise la clé par défaut
func (s *SSHConfig) UsesDefaultKey() bool {
	return s.keyPath == ""
}

// HasCustomKey vérifie si une clé personnalisée est spécifiée
func (s *SSHConfig) HasCustomKey() bool {
	return s.keyPath != ""
}

// KeyExists vérifie si le fichier de clé existe (si spécifié)
func (s *SSHConfig) KeyExists() bool {
	if s.keyPath == "" {
		return true // Pas de clé spécifiée, on assume que l'agent SSH ou la clé par défaut fonctionne
	}

	if _, err := os.Stat(s.keyPath); err != nil {
		return false
	}
	return true
}

// GetKeyFingerprint retourne l'empreinte de la clé (si possible)
func (s *SSHConfig) GetKeyFingerprint() (string, error) {
	if s.keyPath == "" {
		return "", fmt.Errorf("aucun chemin de clé spécifié")
	}

	if !s.KeyExists() {
		return "", fmt.Errorf("fichier de clé inexistant: %s", s.keyPath)
	}

	// En pratique, il faudrait utiliser ssh-keygen pour obtenir l'empreinte
	// Pour l'instant, on retourne juste le chemin
	return fmt.Sprintf("key:%s", filepath.Base(s.keyPath)), nil
}

// Validate effectue une validation complète de la configuration
func (s *SSHConfig) Validate() error {
	return validateSSHConfig(s.host, s.port, s.user, s.keyPath, s.timeout)
}

// MarkAsVerified marque la configuration comme vérifiée
func (s *SSHConfig) MarkAsVerified() *SSHConfig {
	// Retourne une nouvelle instance avec verified=true (immutabilité)
	return &SSHConfig{
		host:     s.host,
		port:     s.port,
		user:     s.user,
		keyPath:  s.keyPath,
		timeout:  s.timeout,
		verified: true,
	}
}

// String implémente fmt.Stringer
func (s *SSHConfig) String() string {
	if s.keyPath != "" {
		return fmt.Sprintf("ssh://%s@%s:%d (key: %s)", s.user, s.host, s.port, filepath.Base(s.keyPath))
	}
	return fmt.Sprintf("ssh://%s@%s:%d", s.user, s.host, s.port)
}

// Equal compare deux configurations SSH
func (s *SSHConfig) Equal(other *SSHConfig) bool {
	if other == nil {
		return false
	}
	return s.host == other.host &&
		s.port == other.port &&
		s.user == other.user &&
		s.keyPath == other.keyPath &&
		s.timeout == other.timeout
}

// ToMap convertit la configuration en map pour la sérialisation
func (s *SSHConfig) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"host":     s.host,
		"port":     s.port,
		"user":     s.user,
		"key_path": s.keyPath,
		"timeout":  s.timeout.String(),
		"verified": s.verified,
	}
}

// validateSSHConfig valide les paramètres de configuration SSH
func validateSSHConfig(host string, port int, user string, keyPath string, timeout time.Duration) error {
	// Validation de l'hôte
	if host == "" {
		return fmt.Errorf("l'hôte SSH ne peut pas être vide")
	}

	if len(host) > 255 {
		return fmt.Errorf("l'hôte SSH est trop long (max 255 caractères)")
	}

	// Validation pour localhost et IP sont OK
	if host != "localhost" && host != "127.0.0.1" && host != "::1" {
		if !hostnamePattern.MatchString(host) {
			return fmt.Errorf("format d'hôte SSH invalide: %s", host)
		}
	}

	// Validation du port
	if port < 1 || port > 65535 {
		return fmt.Errorf("port SSH invalide: %d (doit être entre 1 et 65535)", port)
	}

	// Validation de l'utilisateur
	if user == "" {
		return fmt.Errorf("l'utilisateur SSH ne peut pas être vide")
	}

	if len(user) > 32 {
		return fmt.Errorf("l'utilisateur SSH est trop long (max 32 caractères)")
	}

	if !usernamePattern.MatchString(user) {
		return fmt.Errorf("format d'utilisateur SSH invalide: %s", user)
	}

	// Validation du chemin de clé (optionnel)
	if keyPath != "" {
		if len(keyPath) > 4096 {
			return fmt.Errorf("le chemin de clé SSH est trop long (max 4096 caractères)")
		}

		// Vérifications de sécurité basiques
		if strings.Contains(keyPath, "..") {
			return fmt.Errorf("le chemin de clé SSH ne peut pas contenir '..'")
		}
	}

	// Validation du timeout
	if timeout < 0 {
		return fmt.Errorf("le timeout SSH ne peut pas être négatif")
	}

	if timeout > 10*time.Minute {
		return fmt.Errorf("le timeout SSH est trop long (max 10 minutes)")
	}

	return nil
}
