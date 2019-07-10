package server

import (
	"github.com/hashicorp/errwrap"
	// We must import sha512 so that it registers with the runtime so that
	// certificates that use it can be parsed.
	_ "crypto/sha512"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net"

	"github.com/hashicorp/vault/helper/parseutil"
	"github.com/hashicorp/vault/helper/proxyutil"
	"github.com/hashicorp/vault/helper/reload"
	"github.com/hashicorp/vault/helper/tlsutil"
	"github.com/mitchellh/cli"
)

// ListenerFactory is the factory function to create a listener.
type ListenerFactory func(map[string]interface{}, io.Writer, cli.Ui) (net.Listener, map[string]string, reload.ReloadFunc, error)

// BuiltinListeners is the list of built-in listener types.
var BuiltinListeners = map[string]ListenerFactory{
	"tcp": tcpListenerFactory,
}

// NewListener creates a new listener of the given type with the given
// configuration. The type is looked up in the BuiltinListeners map.
func NewListener(t string, config map[string]interface{}, logger io.Writer, ui cli.Ui) (net.Listener, map[string]string, reload.ReloadFunc, error) {
	f, ok := BuiltinListeners[t]
	if !ok {
		return nil, nil, nil, fmt.Errorf("unknown listener type: %q", t)
	}

	return f(config, logger, ui)
}

func listenerWrapProxy(ln net.Listener, config map[string]interface{}) (net.Listener, error) {
	behaviorRaw, ok := config["proxy_protocol_behavior"]
	if !ok {
		return ln, nil
	}

	behavior, ok := behaviorRaw.(string)
	if !ok {
		return nil, fmt.Errorf("failed parsing proxy_protocol_behavior value: not a string")
	}

	proxyProtoConfig := &proxyutil.ProxyProtoConfig{
		Behavior: behavior,
	}

	if proxyProtoConfig.Behavior == "allow_authorized" || proxyProtoConfig.Behavior == "deny_unauthorized" {
		authorizedAddrsRaw, ok := config["proxy_protocol_authorized_addrs"]
		if !ok {
			return nil, fmt.Errorf("proxy_protocol_behavior set but no proxy_protocol_authorized_addrs value")
		}

		if err := proxyProtoConfig.SetAuthorizedAddrs(authorizedAddrsRaw); err != nil {
			return nil, errwrap.Wrapf("failed parsing proxy_protocol_authorized_addrs: {{err}}", err)
		}
	}

	newLn, err := proxyutil.WrapInProxyProto(ln, proxyProtoConfig)
	if err != nil {
		return nil, errwrap.Wrapf("failed configuring PROXY protocol wrapper: {{err}}", err)
	}

	return newLn, nil
}

func listenerWrapTLS(
	ln net.Listener,
	props map[string]string,
	config map[string]interface{},
	ui cli.Ui) (net.Listener, map[string]string, reload.ReloadFunc, error) {
	props["tls"] = "disabled"

	if v, ok := config["tls_disable"]; ok {
		disabled, err := parseutil.ParseBool(v)
		if err != nil {
			return nil, nil, nil, errwrap.Wrapf("invalid value for 'tls_disable': {{err}}", err)
		}
		if disabled {
			return ln, props, nil, nil
		}
	}

	certFileRaw, ok := config["tls_cert_file"]
	if !ok {
		return nil, nil, nil, fmt.Errorf("'tls_cert_file' must be set")
	}
	certFile := certFileRaw.(string)
	keyFileRaw, ok := config["tls_key_file"]
	if !ok {
		return nil, nil, nil, fmt.Errorf("'tls_key_file' must be set")
	}
	keyFile := keyFileRaw.(string)

	cg := reload.NewCertificateGetter(certFile, keyFile, "")
	if err := cg.Reload(config); err != nil {
		// We try the key without a passphrase first and if we get an incorrect
		// passphrase response, try again after prompting for a passphrase
		if errwrap.Contains(err, x509.IncorrectPasswordError.Error()) {
			var passphrase string
			passphrase, err = ui.AskSecret(fmt.Sprintf("Enter passphrase for %s:", keyFile))
			if err == nil {
				cg = reload.NewCertificateGetter(certFile, keyFile, passphrase)
				if err = cg.Reload(config); err == nil {
					goto PASSPHRASECORRECT
				}
			}
		}
		return nil, nil, nil, errwrap.Wrapf("error loading TLS cert: {{err}}", err)
	}

PASSPHRASECORRECT:
	var tlsvers string
	tlsversRaw, ok := config["tls_min_version"]
	if !ok {
		tlsvers = "tls12"
	} else {
		tlsvers = tlsversRaw.(string)
	}

	tlsConf := &tls.Config{}
	tlsConf.GetCertificate = cg.GetCertificate
	tlsConf.NextProtos = []string{"h2", "http/1.1"}
	tlsConf.MinVersion, ok = tlsutil.TLSLookup[tlsvers]
	if !ok {
		return nil, nil, nil, fmt.Errorf("'tls_min_version' value %q not supported, please specify one of [tls10,tls11,tls12]", tlsvers)
	}
	tlsConf.ClientAuth = tls.RequestClientCert

	if v, ok := config["tls_cipher_suites"]; ok {
		ciphers, err := tlsutil.ParseCiphers(v.(string))
		if err != nil {
			return nil, nil, nil, errwrap.Wrapf("invalid value for 'tls_cipher_suites': {{err}}", err)
		}
		tlsConf.CipherSuites = ciphers
	}
	if v, ok := config["tls_prefer_server_cipher_suites"]; ok {
		preferServer, err := parseutil.ParseBool(v)
		if err != nil {
			return nil, nil, nil, errwrap.Wrapf("invalid value for 'tls_prefer_server_cipher_suites': {{err}}", err)
		}
		tlsConf.PreferServerCipherSuites = preferServer
	}
	var requireVerifyCerts bool
	var err error
	if v, ok := config["tls_require_and_verify_client_cert"]; ok {
		requireVerifyCerts, err = parseutil.ParseBool(v)
		if err != nil {
			return nil, nil, nil, errwrap.Wrapf("invalid value for 'tls_require_and_verify_client_cert': {{err}}", err)
		}
		if requireVerifyCerts {
			tlsConf.ClientAuth = tls.RequireAndVerifyClientCert
		}
		if tlsClientCaFile, ok := config["tls_client_ca_file"]; ok {
			caPool := x509.NewCertPool()
			data, err := ioutil.ReadFile(tlsClientCaFile.(string))
			if err != nil {
				return nil, nil, nil, errwrap.Wrapf("failed to read tls_client_ca_file: {{err}}", err)
			}

			if !caPool.AppendCertsFromPEM(data) {
				return nil, nil, nil, fmt.Errorf("failed to parse CA certificate in tls_client_ca_file")
			}
			tlsConf.ClientCAs = caPool
		}
	}
	if v, ok := config["tls_disable_client_certs"]; ok {
		disableClientCerts, err := parseutil.ParseBool(v)
		if err != nil {
			return nil, nil, nil, errwrap.Wrapf("invalid value for 'tls_disable_client_certs': {{err}}", err)
		}
		if disableClientCerts && requireVerifyCerts {
			return nil, nil, nil, fmt.Errorf("'tls_disable_client_certs' and 'tls_require_and_verify_client_cert' are mutually exclusive")
		}
		if disableClientCerts {
			tlsConf.ClientAuth = tls.NoClientCert
		}
	}

	ln = tls.NewListener(ln, tlsConf)
	props["tls"] = "enabled"
	return ln, props, cg.Reload, nil
}
