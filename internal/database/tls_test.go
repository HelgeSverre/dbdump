package database

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestTLSDSNParam(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		cfg        TLSConfig
		want       string
		wantCustom bool
	}{
		{name: "unset", cfg: TLSConfig{}, want: ""},
		{name: "disabled", cfg: TLSConfig{Mode: TLSDisabled}, want: "false"},
		{name: "preferred", cfg: TLSConfig{Mode: TLSPreferred}, want: "preferred"},
		{name: "require", cfg: TLSConfig{Mode: TLSRequire}, want: "skip-verify"},
		{name: "verify-identity", cfg: TLSConfig{Mode: TLSVerifyIdentity}, want: "true"},
		{name: "skip-verify flag", cfg: TLSConfig{SkipVerify: true}, want: "skip-verify"},
		{name: "verify-ca needs custom", cfg: TLSConfig{Mode: TLSVerifyCA}, wantCustom: true},
		{name: "ca file needs custom", cfg: TLSConfig{Mode: TLSVerifyIdentity, CAFile: "/ca.pem"}, wantCustom: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tlsDSNParam(tc.cfg)
			if err != nil {
				t.Fatalf("tlsDSNParam returned error: %v", err)
			}
			if tc.wantCustom {
				if got == "" || got == "false" || got == "true" || got == "preferred" || got == "skip-verify" {
					t.Fatalf("expected a custom registered name, got %q", got)
				}
				return
			}
			if got != tc.want {
				t.Fatalf("tlsDSNParam(%+v) = %q, want %q", tc.cfg, got, tc.want)
			}
		})
	}
}

func TestTLSConfigValidate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		cfg     TLSConfig
		wantErr bool
	}{
		{name: "empty ok", cfg: TLSConfig{}},
		{name: "valid mode", cfg: TLSConfig{Mode: TLSRequire}},
		{name: "invalid mode", cfg: TLSConfig{Mode: "bogus"}, wantErr: true},
		{name: "cert without key", cfg: TLSConfig{CertFile: "/c.pem"}, wantErr: true},
		{name: "key without cert", cfg: TLSConfig{KeyFile: "/k.pem"}, wantErr: true},
		{name: "cert and key ok", cfg: TLSConfig{CertFile: "/c.pem", KeyFile: "/k.pem"}},
		{name: "skip-verify with verify mode", cfg: TLSConfig{Mode: TLSVerifyIdentity, SkipVerify: true}, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if tc.wantErr != (err != nil) {
				t.Fatalf("Validate(%+v) err=%v, wantErr=%v", tc.cfg, err, tc.wantErr)
			}
		})
	}
}

func TestMySQLDumpTLSArgs(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		cfg     TLSConfig
		sslMode bool
		want    []string
	}{
		{name: "unset", cfg: TLSConfig{}, sslMode: true, want: nil},
		{name: "require with ssl-mode", cfg: TLSConfig{Mode: TLSRequire}, sslMode: true, want: []string{"--ssl-mode=REQUIRED"}},
		{name: "verify-ca with files", cfg: TLSConfig{Mode: TLSVerifyCA, CAFile: "/ca.pem"}, sslMode: true, want: []string{"--ssl-mode=VERIFY_CA", "--ssl-ca=/ca.pem"}},
		{name: "skip-verify downgrades to required", cfg: TLSConfig{SkipVerify: true}, sslMode: true, want: []string{"--ssl-mode=REQUIRED"}},
		{name: "mariadb fallback", cfg: TLSConfig{Mode: TLSRequire}, sslMode: false, want: []string{"--ssl"}},
		{name: "mariadb disabled no flag", cfg: TLSConfig{Mode: TLSDisabled}, sslMode: false, want: nil},
		{name: "mtls all files", cfg: TLSConfig{Mode: TLSVerifyIdentity, CAFile: "/ca.pem", CertFile: "/c.pem", KeyFile: "/k.pem"}, sslMode: true,
			want: []string{"--ssl-mode=VERIFY_IDENTITY", "--ssl-ca=/ca.pem", "--ssl-cert=/c.pem", "--ssl-key=/k.pem"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := mysqlDumpTLSArgs(tc.cfg, tc.sslMode)
			if err != nil {
				t.Fatalf("mysqlDumpTLSArgs returned error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("mysqlDumpTLSArgs(%+v, %v) = %v, want %v", tc.cfg, tc.sslMode, got, tc.want)
			}
		})
	}
}

func TestBuildGoTLSConfigWithCAAndClientCert(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	caPEM, certPEM, keyPEM := generateTestCerts(t)
	caFile := writeTemp(t, dir, "ca.pem", caPEM)
	certFile := writeTemp(t, dir, "client.pem", certPEM)
	keyFile := writeTemp(t, dir, "client-key.pem", keyPEM)

	cfg, err := buildGoTLSConfig(TLSConfig{
		Mode:     TLSVerifyIdentity,
		CAFile:   caFile,
		CertFile: certFile,
		KeyFile:  keyFile,
	})
	if err != nil {
		t.Fatalf("buildGoTLSConfig returned error: %v", err)
	}
	if cfg.RootCAs == nil {
		t.Fatal("expected RootCAs to be populated from the CA file")
	}
	if len(cfg.Certificates) != 1 {
		t.Fatalf("expected one client certificate, got %d", len(cfg.Certificates))
	}
	if cfg.InsecureSkipVerify {
		t.Fatal("verify-identity should not skip verification")
	}
}

func TestBuildGoTLSConfigVerifyCASkipsHostname(t *testing.T) {
	t.Parallel()

	cfg, err := buildGoTLSConfig(TLSConfig{Mode: TLSVerifyCA})
	if err != nil {
		t.Fatalf("buildGoTLSConfig returned error: %v", err)
	}
	if !cfg.InsecureSkipVerify {
		t.Fatal("verify-ca must set InsecureSkipVerify so the driver skips its hostname check")
	}
	if cfg.VerifyConnection == nil {
		t.Fatal("verify-ca must install a custom chain verifier")
	}
}

func TestBuildGoTLSConfigRejectsBadCA(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	bad := writeTemp(t, dir, "bad.pem", []byte("not a certificate"))
	if _, err := buildGoTLSConfig(TLSConfig{Mode: TLSVerifyCA, CAFile: bad}); err == nil {
		t.Fatal("expected an error for a CA file with no certificates")
	}
}

func TestRegisterCustomTLSIsIdempotent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	caPEM, _, _ := generateTestCerts(t)
	caFile := writeTemp(t, dir, "ca.pem", caPEM)
	cfg := TLSConfig{Mode: TLSVerifyCA, CAFile: caFile}

	if err := registerCustomTLS(cfg); err != nil {
		t.Fatalf("registerCustomTLS returned error: %v", err)
	}
	if err := registerCustomTLS(cfg); err != nil {
		t.Fatalf("registerCustomTLS second call returned error: %v", err)
	}
}

func TestRegisterCustomTLSNoopWhenNotConfigured(t *testing.T) {
	t.Parallel()
	if err := registerCustomTLS(TLSConfig{Mode: TLSRequire}); err != nil {
		t.Fatalf("expected no-op for built-in mode, got %v", err)
	}
}

func writeTemp(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	return path
}

// generateTestCerts returns PEM-encoded CA cert, leaf cert (signed by the CA), and
// the leaf private key, all created in-process (no network, no fixtures).
func generateTestCerts(t *testing.T) (caPEM, certPEM, keyPEM []byte) {
	t.Helper()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey (CA) returned error: %v", err)
	}
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "dbdump test CA"},
		NotBefore:             time.Unix(0, 0),
		NotAfter:              time.Unix(1<<31, 0),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("CreateCertificate (CA) returned error: %v", err)
	}

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey (leaf) returned error: %v", err)
	}
	leafTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "dbdump-client"},
		NotBefore:    time.Unix(0, 0),
		NotAfter:     time.Unix(1<<31, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatalf("ParseCertificate (CA) returned error: %v", err)
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTmpl, caCert, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("CreateCertificate (leaf) returned error: %v", err)
	}

	leafKeyDER, err := x509.MarshalPKCS8PrivateKey(leafKey)
	if err != nil {
		t.Fatalf("MarshalPKCS8PrivateKey returned error: %v", err)
	}

	caPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDER})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: leafKeyDER})
	return caPEM, certPEM, keyPEM
}
