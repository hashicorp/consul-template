package dependency

import (
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVaultAgentTokenQuery_Fetch(t *testing.T) {
	t.Parallel()

	// Set up the Vault token file.
	tokenFile, err := ioutil.TempFile("", "token1")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tokenFile.Name())
	ioutil.WriteFile(tokenFile.Name(), []byte("token1"), 0600)

	d, err := NewVaultAgentTokenQuery(tokenFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	clientSet := NewClientSet()
	clientSet.CreateVaultClient(&CreateVaultClientInput{})
	_, _, err = d.Fetch(clientSet, nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "token1", clientSet.Vault().Token())

	// Update the contents.
	ioutil.WriteFile(tokenFile.Name(), []byte("token2"), 0600)
	_, _, err = d.Fetch(clientSet, nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "token2", clientSet.Vault().Token())
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
