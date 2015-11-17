package main

import (
	"bytes"
	"compress/lzw"
	"encoding/gob"
	"log"
	"path"
	"sync"
	"time"

	dep "github.com/hashicorp/consul-template/dependency"
	consulapi "github.com/hashicorp/consul/api"
)

const (
	// sessionCreateRetry is the amount of time we wait
	// to recreate a session when lost.
	sessionCreateRetry = 15 * time.Second

	// lockRetry is the interval on which we try to re-acquire locks
	lockRetry = 10 * time.Second

	// listRetry is the interval on which we retry listing a data path
	listRetry = 10 * time.Second

	// templateDataFlag is added as a flag to the shared data values
	// so that we can use it as a sanity check
	templateDataFlag = 0x22b9a127a2c03520
)

// templateData is GOB encoded share the depdency values
type templateData struct {
	Data map[string]interface{}
}

// DedupManager is used to de-duplicate which instance of Consul-Template
// is handling each template. For each template, a lock path is determined
// using the MD5 of the template. This path is used to elect a "leader"
// instance.
//
// The leader instance operations like usual, but any time a template is
// rendered, any of the data required for rendering is stored in the
// Consul KV store under the lock path.
//
// The follower instances depend on the leader to do the primary watching
// and rendering, and instead only watch the aggregated data in the KV.
// Followers wait for updates and re-render the template.
//
// If a template depends on 50 views, and is running on 50 machines, that
// would normally require 2500 blocking queries. Using deduplication, one
// instance has 50 view queries, plus 50 additional queries on the lock
// path for a total of 100.
//
type DedupManager struct {
	// config is the consul-template configuration
	config *Config

	// clients is used to access the underlying clinets
	clients *dep.ClientSet

	// Brain is where we inject udpates
	brain *Brain

	// templates is the set of templates we are trying to dedup
	templates []*Template

	// leader tracks if we are currently the leader
	leader     map[*Template]<-chan struct{}
	leaderLock sync.RWMutex

	// updateCh is used to indicate an update watched data
	updateCh chan struct{}

	// wg is used to wait for a clean shutdown
	wg sync.WaitGroup

	stop     bool
	stopCh   chan struct{}
	stopLock sync.Mutex
}

// NewDedupManager creates a new Dedup manager
func NewDedupManager(config *Config, clients *dep.ClientSet, brain *Brain, templates []*Template) (*DedupManager, error) {
	d := &DedupManager{
		config:    config,
		clients:   clients,
		brain:     brain,
		templates: templates,
		leader:    make(map[*Template]<-chan struct{}),
		updateCh:  make(chan struct{}, 1),
		stopCh:    make(chan struct{}),
	}
	return d, nil
}

// Start is used to start the de-duplication manager
func (d *DedupManager) Start() error {
	log.Printf("[INFO] (dedup) starting de-duplication manager")

	client, err := d.clients.Consul()
	if err != nil {
		return err
	}
	go d.createSession(client)

	// Start to watch each template
	for _, t := range d.templates {
		go d.watchTemplate(client, t)
	}
	return nil
}

// Stop is used to stop the de-duplication manager
func (d *DedupManager) Stop() error {
	d.stopLock.Lock()
	defer d.stopLock.Unlock()
	if d.stop {
		return nil
	}

	log.Printf("[INFO] (dedup) stopping de-duplication manager")
	d.stop = true
	close(d.stopCh)
	d.wg.Wait()
	return nil
}

// createSession is used to create and maintain a session to Consul
func (d *DedupManager) createSession(client *consulapi.Client) {
START:
	log.Printf("[INFO] (dedup) attempting to create session")
	session := client.Session()
	sessionCh := make(chan struct{})
	se := &consulapi.SessionEntry{
		Name:     "Consul-Template de-duplication",
		Behavior: "delete",
		TTL:      "15s",
	}
	id, _, err := session.Create(se, nil)
	if err != nil {
		log.Printf("[ERR] (dedup) failed to create session: %v", err)
		goto WAIT
	}
	log.Printf("[INFO] (dedup) created session %s", id)

	// Attempt to lock each template
	for _, t := range d.templates {
		d.wg.Add(1)
		go d.attemptLock(client, id, sessionCh, t)
	}

	// Renew our session periodically
	if err := session.RenewPeriodic("15s", id, nil, d.stopCh); err != nil {
		log.Printf("[ERR] (dedup) failed to renew session: %v", err)
	}
	close(sessionCh)

WAIT:
	select {
	case <-time.After(sessionCreateRetry):
		goto START
	case <-d.stopCh:
		return
	}
}

// IsLeader checks if we are currently the leader instance
func (d *DedupManager) IsLeader(tmpl *Template) bool {
	d.leaderLock.RLock()
	defer d.leaderLock.RUnlock()

	lockCh, ok := d.leader[tmpl]
	if !ok {
		return false
	}
	select {
	case <-lockCh:
		return false
	default:
		return true
	}
}

// UpdateDeps is used to update the values of the dependencies for a template
func (d *DedupManager) UpdateDeps(t *Template, deps []dep.Dependency) {
	// Calculate the path to write updates to
	dataPath := path.Join(d.config.Deduplicate.Prefix, t.hexMD5, "data")

	// Package up the dependency data
	td := templateData{
		Data: make(map[string]interface{}),
	}
	for _, dp := range deps {
		// Do not persist any vault related depdendencies
		_, isVaultSecret := dp.(*dep.VaultSecret)
		_, isVaultToken := dp.(*dep.VaultToken)
		if isVaultSecret || isVaultToken {
			continue
		}

		// Pull the current value from the brain
		val, ok := d.brain.Recall(dp)
		if ok {
			td.Data[dp.HashCode()] = val
		}
	}

	// Encode via GOB and LZW compress
	var buf bytes.Buffer
	compress := lzw.NewWriter(&buf, lzw.LSB, 8)
	enc := gob.NewEncoder(compress)
	if err := enc.Encode(&td); err != nil {
		log.Printf("[ERR] (dedup) failed to encode data for '%s': %v",
			dataPath, err)
		return
	}
	compress.Close()

	// Write the KV update
	kvPair := consulapi.KVPair{
		Key:   dataPath,
		Value: buf.Bytes(),
		Flags: templateDataFlag,
	}
	client, err := d.clients.Consul()
	if err != nil {
		log.Printf("[ERR] (dedup) failed to get consul client: %v", err)
		return
	}
	if _, err := client.KV().Put(&kvPair, nil); err != nil {
		log.Printf("[ERR] (dedup) failed to write '%s': %v",
			dataPath, err)
	}
	log.Printf("[INFO] (dedup) updated de-duplicate data '%s'", dataPath)
}

// UpdateCh returns a channel to watch for depedency updates
func (d *DedupManager) UpdateCh() <-chan struct{} {
	return d.updateCh
}

// setLeader sets if we are currently the leader instance
func (d *DedupManager) setLeader(tmpl *Template, lockCh <-chan struct{}) {
	// Update the lock state
	d.leaderLock.Lock()
	if lockCh != nil {
		d.leader[tmpl] = lockCh
	} else {
		delete(d.leader, tmpl)
	}
	d.leaderLock.Unlock()

	// Do an async notify of an update
	select {
	case d.updateCh <- struct{}{}:
	default:
	}
}

func (d *DedupManager) watchTemplate(client *consulapi.Client, t *Template) {
	log.Printf("[INFO] (dedup) starting watch for template hash %s", t.hexMD5)
	path := path.Join(d.config.Deduplicate.Prefix, t.hexMD5, "data")
	opts := &consulapi.QueryOptions{WaitTime: 60 * time.Second}
START:
	// Stop listening if we're stopped
	select {
	case <-d.stopCh:
		return
	default:
	}

	// If we are current the leader, wait for leadership lost
	d.leaderLock.RLock()
	lockCh, ok := d.leader[t]
	d.leaderLock.RUnlock()
	if ok {
		select {
		case <-lockCh:
			goto START
		case <-d.stopCh:
			return
		}
	}

	// Block for updates on the data key
	log.Printf("[INFO] (dedup) listing data for template hash %s", t.hexMD5)
	pair, meta, err := client.KV().Get(path, opts)
	if err != nil {
		log.Printf("[ERR] (dedup) failed to get '%s': %v", path, err)
		select {
		case <-time.After(listRetry):
			goto START
		case <-d.stopCh:
			return
		}
	}
	opts.WaitIndex = meta.LastIndex

	// Stop listening if we're stopped
	select {
	case <-d.stopCh:
		return
	default:
	}

	// If we are current the leader, wait for leadership lost
	d.leaderLock.RLock()
	lockCh, ok = d.leader[t]
	d.leaderLock.RUnlock()
	if ok {
		select {
		case <-lockCh:
			goto START
		case <-d.stopCh:
			return
		}
	}

	// Parse the data file
	if pair != nil && pair.Flags == templateDataFlag {
		d.parseData(pair.Key, pair.Value)
	}
	goto START
}

// parseData is used to update brain from a KV data pair
func (d *DedupManager) parseData(path string, raw []byte) {
	// Setup the decompression and decoders
	r := bytes.NewReader(raw)
	decompress := lzw.NewReader(r, lzw.LSB, 8)
	defer decompress.Close()
	dec := gob.NewDecoder(decompress)

	// Decode the data
	var td templateData
	if err := dec.Decode(&td); err != nil {
		log.Printf("[ERR] (dedup) failed to decode '%s': %v",
			path, err)
		return
	}
	log.Printf("[INFO] (dedup) loading %d dependencies from '%s'",
		len(td.Data), path)

	// Update the data in the brain
	for hashCode, value := range td.Data {
		d.brain.ForceSet(hashCode, value)
	}

	// Trigger the updateCh
	select {
	case d.updateCh <- struct{}{}:
	default:
	}
}

func (d *DedupManager) attemptLock(client *consulapi.Client, session string, sessionCh chan struct{}, t *Template) {
	defer d.wg.Done()
START:
	log.Printf("[INFO] (dedup) attempting lock for template hash %s", t.hexMD5)
	basePath := path.Join(d.config.Deduplicate.Prefix, t.hexMD5)
	lopts := &consulapi.LockOptions{
		Key:     path.Join(basePath, "lock"),
		Session: session,
	}
	lock, err := client.LockOpts(lopts)
	if err != nil {
		log.Printf("[ERR] (dedup) failed to create lock '%s': %v",
			lopts.Key, err)
		return
	}

	var retryCh <-chan time.Time
	leaderCh, err := lock.Lock(sessionCh)
	if err != nil {
		log.Printf("[ERR] (dedup) failed to acquire lock '%s': %v",
			lopts.Key, err)
		retryCh = time.After(lockRetry)
	} else {
		log.Printf("[INFO] (dedup) acquired lock '%s'", lopts.Key)
		d.setLeader(t, leaderCh)
	}

	select {
	case <-retryCh:
		retryCh = nil
		goto START
	case <-leaderCh:
		log.Printf("[WARN] (dedup) lost lock ownership '%s'", lopts.Key)
		d.setLeader(t, nil)
		goto START
	case <-sessionCh:
		log.Printf("[INFO] (dedup) releasing lock '%s'", lopts.Key)
		d.setLeader(t, nil)
		lock.Unlock()
	case <-d.stopCh:
		log.Printf("[INFO] (dedup) releasing lock '%s'", lopts.Key)
		lock.Unlock()
	}
}
