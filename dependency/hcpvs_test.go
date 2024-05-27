package dependency

import (
	"testing"

	secretspreview "github.com/hashicorp/hcp-sdk-go/clients/cloud-vault-secrets/preview/2023-11-28/client/secret_service"
)

func TestHCPVS(t *testing.T) {
	appName := t.Name()
	hcpvs := testHCPVS(t, appName)
	p := secretspreview.NewListAppSecretsParams()
	p.AppName = appName
	_, err := hcpvs.hcpvs.client.ListAppSecrets(p, nil)
	if err != nil {
		t.Fatal(err)
	}

}
