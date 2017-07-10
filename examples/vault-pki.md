Rendering PKI Certificates from Vault with Consul Template
----------------------------------------------------------
[Vault][vault] is a popular open source tool for managing secrets. In addition
to acting as an encrypted KV store, Vault can also generate dynamic secrets,
like PKI/TLS certificates.

When generating PKI certificates with Vault, the certificate, private key, and
any intermediate certs are all returned as part of the same API call. Most
software requires these files be placed in separate files on the system.

[vault]: https://www.vaultproject.io/ "Vault by HashiCorp"

## Multiple Output Files

Consul Template can run more than one template. At boot, all dependencies
(external API requests) are mapped into a single list. This means that multiple
templates watching the same path return the same data.

Consider the following two templates:

```liquid
{{- # /tmp/cert.tpl -}}
{{ with secret "pki/issue/my-domain-dot-com" "common_name=foo.example.com" }}
{{ .Data.certificate }}{{ end }}
```

```liquid
{{- # /tmp/ca.tpl -}}
{{ with secret "pki/issue/my-domain-dot-com" "common_name=foo.example.com" }}
{{ .Data.ca_chain }}{{ end }}
```

```liquid
{{- # /tmp/key.tpl -}}
{{ with secret "pki/issue/my-domain-dot-com" "common_name=foo.example.com" }}
{{ .Data.private_key }}{{ end }}
```

These are three different input templates, but when run under the same Consul
Template process, they are compressed into a single API call, sharing the
resulting data.

Here is an example Consul Template configuration:

```hcl
template {
  source      = "/tmp/cert.tpl"
  destination = "/opt/my-app/ssl/my-app.crt"
}

template {
  source      = "/tmp/ca.tpl"
  destination = "/opt/my-app/ssl/ca.crt"
}

template {
  source      = "/tmp/key.tpl"
  destination = "/opt/my-app/ssl/my-app.key"
}
```

To generate multiple certificates of the same path, use multiple Consul Template
processes.
