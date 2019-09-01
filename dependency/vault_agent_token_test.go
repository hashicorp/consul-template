package dependency

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/consul-template/renderer"
	"github.com/stretchr/testify/assert"
)

func TestVaultAgentTokenQuery_Fetch(t *testing.T) {
	// Don't use t.Parallel() here as the SetToken() calls are global and break
	// other tests if run in parallel

	// reset token back to original
	vc := testClients.Vault()
	token := vc.Token()
	defer vc.SetToken(token)

	// Set up the Vault token file.
	tokenFile, err := ioutil.TempFile("", "token1")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tokenFile.Name())
	renderer.AtomicWrite(tokenFile.Name(), false, []byte("token"), 0644, false, 0)

	d, err := NewVaultAgentTokenQuery(tokenFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	clientSet := testClients
	_, _, err = d.Fetch(clientSet, nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "token", clientSet.Vault().Token())

	// Update the contents.
	renderer.AtomicWrite(tokenFile.Name(), false, []byte("another_token"), 0644, false, 0)
	_, _, err = d.Fetch(clientSet, nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "another_token", clientSet.Vault().Token())
}

func TestVaultAgentTokenQuery_Fetch_missingFile(t *testing.T) {
	t.Parallel()

	// Use a non-existant token file path.
	d, err := NewVaultAgentTokenQuery("/tmp/invalid-file")
	if err != nil {
		t.Fatal(err)
	}

	clientSet := NewClientSet()
	clientSet.CreateVaultClient(&CreateVaultClientInput{
		Token: "foo",
	})
	_, _, err = d.Fetch(clientSet, nil)
	if err == nil || !strings.Contains(err.Error(), "no such file") {
		t.Fatal(err)
	}

	// Token should be unaffected.
	assert.Equal(t, "foo", clientSet.Vault().Token())
}
