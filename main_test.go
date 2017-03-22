package main

import (
	"io/ioutil"
	"log"
	"os"
	"testing"

	dep "github.com/hashicorp/consul-template/dependency"
	"github.com/hashicorp/consul/testutil"
)

var testConsul *testutil.TestServer
var testClients *dep.ClientSet

func TestMain(m *testing.M) {
	consul, err := testutil.NewTestServerConfig(func(c *testutil.TestServerConfig) {
		c.LogLevel = "warn"
		c.Stdout = ioutil.Discard
		c.Stderr = ioutil.Discard
	})
	if err != nil {
		log.Fatal("failed to start consul server")
	}
	testConsul = consul

	clients := dep.NewClientSet()
	if err := clients.CreateConsulClient(&dep.CreateConsulClientInput{
		Address: testConsul.HTTPAddr,
	}); err != nil {
		testConsul.Stop()
		log.Fatal(err)
	}
	testClients = clients

	exitCh := make(chan int, 1)
	func() {
		defer func() {
			// Attempt to recover from a panic and stop the server. If we don't stop
			// it, the panic will cause the server to remain running in the
			// background. Here we catch the panic and the re-raise it.
			if r := recover(); r != nil {
				testConsul.Stop()
				panic(r)
			}
		}()

		exitCh <- m.Run()
	}()

	exit := <-exitCh

	testConsul.Stop()
	os.Exit(exit)
}
