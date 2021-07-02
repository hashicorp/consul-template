consul {
  address = "{{ .consulAddress }}"
}

vault {
  address = "{{ .vaultAddress }}"
  token =  "{{ .vaultToken }}"
}

log_level = "trace"

template {
  source = "examples/aws/apis.yml.ctmpl"
  destination = "./risk_management-apis.yml"
}