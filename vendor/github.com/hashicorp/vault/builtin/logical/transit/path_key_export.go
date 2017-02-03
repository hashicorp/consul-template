package transit

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/vault/helper/keysutil"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

const (
	exportTypeEncryptionKey = "encryption-key"
	exportTypeSigningKey    = "signing-key"
	exportTypeHMACKey       = "hmac-key"
)

func (b *backend) pathExportKeys() *framework.Path {
	return &framework.Path{
		Pattern: "export/" + framework.GenericNameRegex("export_type") + "/" + framework.GenericNameRegex("name") + framework.OptionalParamRegex("version"),
		Fields: map[string]*framework.FieldSchema{
			"export_type": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "Type of key to export (encryption-key, signing-key, hmac-key)",
			},
			"name": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "Name of the key",
			},
			"version": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "Version of the key",
			},
		},

		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.ReadOperation: b.pathPolicyExportRead,
		},

		HelpSynopsis:    pathExportHelpSyn,
		HelpDescription: pathExportHelpDesc,
	}
}

func (b *backend) pathPolicyExportRead(
	req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	exportType := d.Get("export_type").(string)
	name := d.Get("name").(string)
	version := d.Get("version").(string)

	switch exportType {
	case exportTypeEncryptionKey:
	case exportTypeSigningKey:
	case exportTypeHMACKey:
	default:
		return logical.ErrorResponse(fmt.Sprintf("invalid export type: %s", exportType)), logical.ErrInvalidRequest
	}

	p, lock, err := b.lm.GetPolicyShared(req.Storage, name)
	if lock != nil {
		defer lock.RUnlock()
	}
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, nil
	}

	if !p.Exportable {
		return logical.ErrorResponse("key is not exportable"), nil
	}

	switch exportType {
	case exportTypeEncryptionKey:
		if !p.Type.EncryptionSupported() {
			return logical.ErrorResponse("encryption not supported for the key"), logical.ErrInvalidRequest
		}
	case exportTypeSigningKey:
		if !p.Type.SigningSupported() {
			return logical.ErrorResponse("signing not supported for the key"), logical.ErrInvalidRequest
		}
	}

	resp := &logical.Response{
		Data: map[string]interface{}{
			"name": p.Name,
		},
	}

	if version != "" {
		var versionValue int
		if version == "latest" {
			versionValue = p.LatestVersion
		} else {
			version = strings.TrimPrefix(version, "v")
			versionValue, err = strconv.Atoi(version)
			if err != nil {
				return logical.ErrorResponse("invalid key version"), logical.ErrInvalidRequest
			}
		}
		resp.Data["version"] = versionValue

		key, ok := p.Keys[versionValue]
		if !ok {
			return logical.ErrorResponse("version does not exist or is no longer valid"), logical.ErrInvalidRequest
		}

		exportKey, err := getExportKey(p, &key, exportType)
		if err != nil {
			return nil, err
		}
		resp.Data["key"] = exportKey

		return resp, nil
	}

	retKeys := map[string]string{}
	for k, v := range p.Keys {
		exportKey, err := getExportKey(p, &v, exportType)
		if err != nil {
			return nil, err
		}
		retKeys[strconv.Itoa(k)] = exportKey
	}
	resp.Data["keys"] = retKeys

	return resp, nil
}

func getExportKey(policy *keysutil.Policy, key *keysutil.KeyEntry, exportType string) (string, error) {
	if policy == nil {
		return "", errors.New("nil policy provided")
	}

	switch exportType {
	case exportTypeHMACKey:
		return strings.TrimSpace(base64.StdEncoding.EncodeToString(key.HMACKey)), nil
	case exportTypeEncryptionKey:
		switch policy.Type {
		case keysutil.KeyType_AES256_GCM96:
			return strings.TrimSpace(base64.StdEncoding.EncodeToString(key.AESKey)), nil
		}
	case exportTypeSigningKey:
		switch policy.Type {
		case keysutil.KeyType_ECDSA_P256:
			ecKey, err := keyEntryToECPrivateKey(key, elliptic.P256())
			if err != nil {
				return "", err
			}
			return ecKey, nil
		}
	}

	return "", fmt.Errorf("unknown key type %v", policy.Type)
}

func keyEntryToECPrivateKey(k *keysutil.KeyEntry, curve elliptic.Curve) (string, error) {
	if k == nil {
		return "", errors.New("nil KeyEntry provided")
	}

	privKey := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: curve,
			X:     k.EC_X,
			Y:     k.EC_Y,
		},
		D: k.EC_D,
	}
	ecder, err := x509.MarshalECPrivateKey(privKey)
	if err != nil {
		return "", err
	}
	if ecder == nil {
		return "", errors.New("No data returned when marshalling to private key")
	}

	block := pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: ecder,
	}
	return strings.TrimSpace(string(pem.EncodeToMemory(&block))), nil
}

const pathExportHelpSyn = `Export named encryption or signing key`

const pathExportHelpDesc = `
This path is used to export the named keys that are configured as
exportable.
`
