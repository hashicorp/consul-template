package transit

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/hashicorp/vault/helper/errutil"
	"github.com/hashicorp/vault/helper/keysutil"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

func (b *backend) pathSign() *framework.Path {
	return &framework.Path{
		Pattern: "sign/" + framework.GenericNameRegex("name") + framework.OptionalParamRegex("urlalgorithm"),
		Fields: map[string]*framework.FieldSchema{
			"name": {
				Type:        framework.TypeString,
				Description: "The key to use",
			},

			"input": {
				Type:        framework.TypeString,
				Description: "The base64-encoded input data",
			},

			"context": {
				Type: framework.TypeString,
				Description: `Base64 encoded context for key derivation. Required if key
derivation is enabled; currently only available with ed25519 keys.`,
			},

			"hash_algorithm": {
				Type:    framework.TypeString,
				Default: "sha2-256",
				Description: `Hash algorithm to use (POST body parameter). Valid values are:

* sha1
* sha2-224
* sha2-256
* sha2-384
* sha2-512

Defaults to "sha2-256". Not valid for all key types,
including ed25519.`,
			},

			"algorithm": {
				Type:        framework.TypeString,
				Default:     "sha2-256",
				Description: `Deprecated: use "hash_algorithm" instead.`,
			},

			"urlalgorithm": {
				Type:        framework.TypeString,
				Description: `Hash algorithm to use (POST URL parameter)`,
			},

			"key_version": {
				Type: framework.TypeInt,
				Description: `The version of the key to use for signing.
Must be 0 (for latest) or a value greater than or equal
to the min_encryption_version configured on the key.`,
			},

			"prehashed": {
				Type:        framework.TypeBool,
				Description: `Set to 'true' when the input is already hashed. If the key type is 'rsa-2048' or 'rsa-4096', then the algorithm used to hash the input should be indicated by the 'algorithm' parameter.`,
			},

			"signature_algorithm": {
				Type: framework.TypeString,
				Description: `The signature algorithm to use for signing. Currently only applies to RSA key types.
Options are 'pss' or 'pkcs1v15'. Defaults to 'pss'`,
			},

			"marshaling_algorithm": {
				Type:        framework.TypeString,
				Default:     "asn1",
				Description: `The method by which to marshal the signature. The default is 'asn1' which is used by openssl and X.509. It can also be set to 'jws' which is used for JWT signatures; setting it to this will also cause the encoding of the signature to be url-safe base64 instead of using standard base64 encoding. Currently only valid for ECDSA P-256 key types".`,
			},
		},

		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.UpdateOperation: b.pathSignWrite,
		},

		HelpSynopsis:    pathSignHelpSyn,
		HelpDescription: pathSignHelpDesc,
	}
}

func (b *backend) pathVerify() *framework.Path {
	return &framework.Path{
		Pattern: "verify/" + framework.GenericNameRegex("name") + framework.OptionalParamRegex("urlalgorithm"),
		Fields: map[string]*framework.FieldSchema{
			"name": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "The key to use",
			},

			"context": {
				Type: framework.TypeString,
				Description: `Base64 encoded context for key derivation. Required if key
derivation is enabled; currently only available with ed25519 keys.`,
			},

			"signature": {
				Type:        framework.TypeString,
				Description: "The signature, including vault header/key version",
			},

			"hmac": {
				Type:        framework.TypeString,
				Description: "The HMAC, including vault header/key version",
			},

			"input": {
				Type:        framework.TypeString,
				Description: "The base64-encoded input data to verify",
			},

			"urlalgorithm": {
				Type:        framework.TypeString,
				Description: `Hash algorithm to use (POST URL parameter)`,
			},

			"hash_algorithm": {
				Type:    framework.TypeString,
				Default: "sha2-256",
				Description: `Hash algorithm to use (POST body parameter). Valid values are:

* sha1
* sha2-224
* sha2-256
* sha2-384
* sha2-512

Defaults to "sha2-256". Not valid for all key types.`,
			},

			"algorithm": {
				Type:        framework.TypeString,
				Default:     "sha2-256",
				Description: `Deprecated: use "hash_algorithm" instead.`,
			},

			"prehashed": {
				Type:        framework.TypeBool,
				Description: `Set to 'true' when the input is already hashed. If the key type is 'rsa-2048' or 'rsa-4096', then the algorithm used to hash the input should be indicated by the 'algorithm' parameter.`,
			},

			"signature_algorithm": {
				Type: framework.TypeString,
				Description: `The signature algorithm to use for signature verification. Currently only applies to RSA key types. 
Options are 'pss' or 'pkcs1v15'. Defaults to 'pss'`,
			},

			"marshaling_algorithm": {
				Type:        framework.TypeString,
				Default:     "asn1",
				Description: `The method by which to unmarshal the signature when verifying. The default is 'asn1' which is used by openssl and X.509; can also be set to 'jws' which is used for JWT signatures in which case the signature is also expected to be url-safe base64 encoding instead of standard base64 encoding. Currently only valid for ECDSA P-256 key types".`,
			},
		},

		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.UpdateOperation: b.pathVerifyWrite,
		},

		HelpSynopsis:    pathVerifyHelpSyn,
		HelpDescription: pathVerifyHelpDesc,
	}
}

func (b *backend) pathSignWrite(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	name := d.Get("name").(string)
	ver := d.Get("key_version").(int)
	inputB64 := d.Get("input").(string)
	hashAlgorithmStr := d.Get("urlalgorithm").(string)
	if hashAlgorithmStr == "" {
		hashAlgorithmStr = d.Get("hash_algorithm").(string)
		if hashAlgorithmStr == "" {
			hashAlgorithmStr = d.Get("algorithm").(string)
		}
	}

	hashAlgorithm, ok := keysutil.HashTypeMap[hashAlgorithmStr]
	if !ok {
		return logical.ErrorResponse(fmt.Sprintf("invalid hash algorithm %q", hashAlgorithmStr)), logical.ErrInvalidRequest
	}

	marshalingStr := d.Get("marshaling_algorithm").(string)
	marshaling, ok := keysutil.MarshalingTypeMap[marshalingStr]
	if !ok {
		return logical.ErrorResponse(fmt.Sprintf("invalid marshaling type %q", marshalingStr)), logical.ErrInvalidRequest
	}

	prehashed := d.Get("prehashed").(bool)
	sigAlgorithm := d.Get("signature_algorithm").(string)

	input, err := base64.StdEncoding.DecodeString(inputB64)
	if err != nil {
		return logical.ErrorResponse(fmt.Sprintf("unable to decode input as base64: %s", err)), logical.ErrInvalidRequest
	}

	// Get the policy
	p, _, err := b.lm.GetPolicy(ctx, keysutil.PolicyRequest{
		Storage: req.Storage,
		Name:    name,
	})
	if err != nil {
		return nil, err
	}
	if p == nil {
		return logical.ErrorResponse("encryption key not found"), logical.ErrInvalidRequest
	}
	if !b.System().CachingDisabled() {
		p.Lock(false)
	}

	if !p.Type.SigningSupported() {
		p.Unlock()
		return logical.ErrorResponse(fmt.Sprintf("key type %v does not support signing", p.Type)), logical.ErrInvalidRequest
	}

	contextRaw := d.Get("context").(string)
	var context []byte
	if len(contextRaw) != 0 {
		context, err = base64.StdEncoding.DecodeString(contextRaw)
		if err != nil {
			p.Unlock()
			return logical.ErrorResponse("failed to base64-decode context"), logical.ErrInvalidRequest
		}
	}

	if p.Type.HashSignatureInput() && !prehashed {
		hf := keysutil.HashFuncMap[hashAlgorithm]()
		hf.Write(input)
		input = hf.Sum(nil)
	}

	sig, err := p.Sign(ver, context, input, hashAlgorithm, sigAlgorithm, marshaling)
	if err != nil {
		p.Unlock()
		return nil, err
	}
	if sig == nil {
		p.Unlock()
		return nil, fmt.Errorf("signature could not be computed")
	}

	// Generate the response
	resp := &logical.Response{
		Data: map[string]interface{}{
			"signature": sig.Signature,
		},
	}

	if len(sig.PublicKey) > 0 {
		resp.Data["public_key"] = sig.PublicKey
	}

	p.Unlock()
	return resp, nil
}

func (b *backend) pathVerifyWrite(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	sig := d.Get("signature").(string)
	hmac := d.Get("hmac").(string)
	switch {
	case sig != "" && hmac != "":
		return logical.ErrorResponse("provide one of 'signature' or 'hmac'"), logical.ErrInvalidRequest

	case sig == "" && hmac == "":
		return logical.ErrorResponse("neither a 'signature' nor an 'hmac' were given to verify"), logical.ErrInvalidRequest

	case hmac != "":
		return b.pathHMACVerify(ctx, req, d, hmac)
	}

	name := d.Get("name").(string)
	inputB64 := d.Get("input").(string)
	hashAlgorithmStr := d.Get("urlalgorithm").(string)
	if hashAlgorithmStr == "" {
		hashAlgorithmStr = d.Get("hash_algorithm").(string)
		if hashAlgorithmStr == "" {
			hashAlgorithmStr = d.Get("algorithm").(string)
		}
	}

	hashAlgorithm, ok := keysutil.HashTypeMap[hashAlgorithmStr]
	if !ok {
		return logical.ErrorResponse(fmt.Sprintf("invalid hash algorithm %q", hashAlgorithmStr)), logical.ErrInvalidRequest
	}

	marshalingStr := d.Get("marshaling_algorithm").(string)
	marshaling, ok := keysutil.MarshalingTypeMap[marshalingStr]
	if !ok {
		return logical.ErrorResponse(fmt.Sprintf("invalid marshaling type %q", marshalingStr)), logical.ErrInvalidRequest
	}

	prehashed := d.Get("prehashed").(bool)
	sigAlgorithm := d.Get("signature_algorithm").(string)

	input, err := base64.StdEncoding.DecodeString(inputB64)
	if err != nil {
		return logical.ErrorResponse(fmt.Sprintf("unable to decode input as base64: %s", err)), logical.ErrInvalidRequest
	}

	// Get the policy
	p, _, err := b.lm.GetPolicy(ctx, keysutil.PolicyRequest{
		Storage: req.Storage,
		Name:    name,
	})
	if err != nil {
		return nil, err
	}
	if p == nil {
		return logical.ErrorResponse("encryption key not found"), logical.ErrInvalidRequest
	}
	if !b.System().CachingDisabled() {
		p.Lock(false)
	}

	if !p.Type.SigningSupported() {
		p.Unlock()
		return logical.ErrorResponse(fmt.Sprintf("key type %v does not support verification", p.Type)), logical.ErrInvalidRequest
	}

	contextRaw := d.Get("context").(string)
	var context []byte
	if len(contextRaw) != 0 {
		context, err = base64.StdEncoding.DecodeString(contextRaw)
		if err != nil {
			p.Unlock()
			return logical.ErrorResponse("failed to base64-decode context"), logical.ErrInvalidRequest
		}
	}

	if p.Type.HashSignatureInput() && !prehashed {
		hf := keysutil.HashFuncMap[hashAlgorithm]()
		hf.Write(input)
		input = hf.Sum(nil)
	}

	valid, err := p.VerifySignature(context, input, hashAlgorithm, sigAlgorithm, marshaling, sig)
	if err != nil {
		switch err.(type) {
		case errutil.UserError:
			p.Unlock()
			return logical.ErrorResponse(err.Error()), logical.ErrInvalidRequest
		case errutil.InternalError:
			p.Unlock()
			return nil, err
		default:
			p.Unlock()
			return nil, err
		}
	}

	// Generate the response
	resp := &logical.Response{
		Data: map[string]interface{}{
			"valid": valid,
		},
	}

	p.Unlock()
	return resp, nil
}

const pathSignHelpSyn = `Generate a signature for input data using the named key`

const pathSignHelpDesc = `
Generates a signature of the input data using the named key and the given hash algorithm.
`
const pathVerifyHelpSyn = `Verify a signature or HMAC for input data created using the named key`

const pathVerifyHelpDesc = `
Verifies a signature or HMAC of the input data using the named key and the given hash algorithm.
`
