package database

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"hash/fnv"
	"os"
	"strconv"
	"strings"

	"github.com/go-sql-driver/mysql"
)

// TLS modes, mirroring MySQL's ssl-mode semantics.
const (
	TLSDisabled       = "disabled"        // no encryption
	TLSPreferred      = "preferred"       // encrypt if the server supports it, else plaintext
	TLSRequire        = "require"         // encrypt, but do not verify the server certificate
	TLSVerifyCA       = "verify-ca"       // encrypt and verify the cert chain (not the hostname)
	TLSVerifyIdentity = "verify-identity" // encrypt and verify the chain and hostname
)

// TLSConfig describes how to secure the connection to the database. An empty,
// zero value means "not configured": dbdump behaves exactly as before (no TLS
// wiring on either the Go connection or the mysqldump subprocess).
type TLSConfig struct {
	Mode       string
	SkipVerify bool
	CAFile     string
	CertFile   string
	KeyFile    string
	ServerName string
}

// resolveMode returns the effective, validated mode. An empty return means TLS is
// not configured. A mode is inferred when only files or --tls-skip-verify are set.
func (t TLSConfig) resolveMode() (string, error) {
	mode := strings.ToLower(strings.TrimSpace(t.Mode))
	if mode == "" {
		switch {
		case t.SkipVerify:
			mode = TLSRequire
		case t.CAFile != "" || t.CertFile != "" || t.KeyFile != "" || t.ServerName != "":
			mode = TLSVerifyCA
		default:
			return "", nil
		}
	}

	switch mode {
	case TLSDisabled, TLSPreferred, TLSRequire, TLSVerifyCA, TLSVerifyIdentity:
		return mode, nil
	default:
		return "", fmt.Errorf("invalid TLS mode %q (expected disabled, preferred, require, verify-ca, or verify-identity)", t.Mode)
	}
}

// Validate reports configuration errors without touching the filesystem.
func (t TLSConfig) Validate() error {
	mode, err := t.resolveMode()
	if err != nil {
		return err
	}
	if (t.CertFile == "") != (t.KeyFile == "") {
		return errors.New("--tls-cert and --tls-key must be provided together")
	}
	if t.SkipVerify && (mode == TLSVerifyCA || mode == TLSVerifyIdentity) {
		return fmt.Errorf("--tls-skip-verify cannot be combined with --tls-mode %s", mode)
	}
	return nil
}

// needsCustomTLS reports whether the go-sql-driver needs a registered *tls.Config
// rather than a built-in token. verify-ca always needs one (the driver has no
// built-in "verify chain but not hostname" mode), as does anything with a custom
// CA, client cert, or server name.
func (t TLSConfig) needsCustomTLS(mode string) bool {
	if t.CAFile != "" || t.CertFile != "" || t.KeyFile != "" || t.ServerName != "" {
		return true
	}
	if t.SkipVerify {
		return false
	}
	return mode == TLSVerifyCA
}

// customTLSName is a deterministic registration key derived from the config, so
// repeated calls (and parallel tests) reuse one entry instead of leaking many.
func customTLSName(t TLSConfig) string {
	key := strings.Join([]string{
		t.Mode, strconv.FormatBool(t.SkipVerify), t.CAFile, t.CertFile, t.KeyFile, t.ServerName,
	}, "|")
	h := fnv.New64a()
	_, _ = h.Write([]byte(key))
	return fmt.Sprintf("dbdump-%x", h.Sum64())
}

// tlsDSNParam returns the value for the go-sql-driver `tls` DSN parameter. An
// empty string means "add no tls parameter" (unchanged, plaintext behavior).
func tlsDSNParam(t TLSConfig) (string, error) {
	mode, err := t.resolveMode()
	if err != nil {
		return "", err
	}
	if mode == "" {
		return "", nil
	}
	if t.needsCustomTLS(mode) {
		return customTLSName(t), nil
	}
	if t.SkipVerify {
		return "skip-verify", nil
	}

	switch mode {
	case TLSDisabled:
		return "false", nil
	case TLSPreferred:
		return "preferred", nil
	case TLSRequire:
		return "skip-verify", nil
	case TLSVerifyIdentity:
		return "true", nil
	default:
		return "", nil
	}
}

// buildGoTLSConfig builds the *tls.Config for the cases that need a custom one.
func buildGoTLSConfig(t TLSConfig) (*tls.Config, error) {
	mode, err := t.resolveMode()
	if err != nil {
		return nil, err
	}

	cfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if t.ServerName != "" {
		cfg.ServerName = t.ServerName
	}

	var rootCAs *x509.CertPool
	if t.CAFile != "" {
		pemData, err := os.ReadFile(t.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read TLS CA file: %w", err)
		}
		rootCAs = x509.NewCertPool()
		if !rootCAs.AppendCertsFromPEM(pemData) {
			return nil, fmt.Errorf("no certificates found in TLS CA file %q", t.CAFile)
		}
		cfg.RootCAs = rootCAs
	}

	if t.CertFile != "" || t.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(t.CertFile, t.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS client certificate: %w", err)
		}
		cfg.Certificates = []tls.Certificate{cert}
	}

	switch {
	case t.SkipVerify:
		cfg.InsecureSkipVerify = true
	case mode == TLSVerifyCA:
		// Verify the chain against the roots but skip the hostname check.
		cfg.InsecureSkipVerify = true
		cfg.VerifyConnection = func(cs tls.ConnectionState) error {
			opts := x509.VerifyOptions{
				Roots:         rootCAs, // nil => system roots
				Intermediates: x509.NewCertPool(),
			}
			for _, cert := range cs.PeerCertificates[1:] {
				opts.Intermediates.AddCert(cert)
			}
			_, err := cs.PeerCertificates[0].Verify(opts)
			return err
		}
	}

	return cfg, nil
}

// registerCustomTLS registers the custom *tls.Config under its deterministic name
// so the DSN's `tls=<name>` parameter resolves. It is a no-op when TLS is not
// configured or a built-in token suffices.
func registerCustomTLS(t TLSConfig) error {
	mode, err := t.resolveMode()
	if err != nil {
		return err
	}
	if mode == "" || !t.needsCustomTLS(mode) {
		return nil
	}

	cfg, err := buildGoTLSConfig(t)
	if err != nil {
		return err
	}
	if err := mysql.RegisterTLSConfig(customTLSName(t), cfg); err != nil {
		return fmt.Errorf("failed to register TLS config: %w", err)
	}
	return nil
}

// mysqlDumpTLSArgs maps the TLS config onto mysqldump command-line flags. When the
// client lacks --ssl-mode (e.g. MariaDB's mariadb-dump) it falls back to --ssl.
func mysqlDumpTLSArgs(t TLSConfig, sslModeSupported bool) ([]string, error) {
	mode, err := t.resolveMode()
	if err != nil {
		return nil, err
	}
	if mode == "" {
		return nil, nil
	}
	if t.SkipVerify {
		mode = TLSRequire
	}

	var args []string
	if sslModeSupported {
		sslMode := map[string]string{
			TLSDisabled:       "DISABLED",
			TLSPreferred:      "PREFERRED",
			TLSRequire:        "REQUIRED",
			TLSVerifyCA:       "VERIFY_CA",
			TLSVerifyIdentity: "VERIFY_IDENTITY",
		}[mode]
		args = append(args, "--ssl-mode="+sslMode)
	} else if mode != TLSDisabled && mode != TLSPreferred {
		// Older clients without --ssl-mode: just request encryption.
		args = append(args, "--ssl")
	}

	if t.CAFile != "" {
		args = append(args, "--ssl-ca="+t.CAFile)
	}
	if t.CertFile != "" {
		args = append(args, "--ssl-cert="+t.CertFile)
	}
	if t.KeyFile != "" {
		args = append(args, "--ssl-key="+t.KeyFile)
	}

	return args, nil
}
