package plugin

import (
	"context"
	"net/rpc"
	"time"

	"fmt"

	"github.com/hashicorp/vault/helper/consts"
	"github.com/hashicorp/vault/helper/license"
	"github.com/hashicorp/vault/helper/pluginutil"
	"github.com/hashicorp/vault/helper/wrapping"
	"github.com/hashicorp/vault/logical"
)

type SystemViewClient struct {
	client *rpc.Client
}

func (s *SystemViewClient) DefaultLeaseTTL() time.Duration {
	var reply DefaultLeaseTTLReply
	err := s.client.Call("Plugin.DefaultLeaseTTL", new(interface{}), &reply)
	if err != nil {
		return 0
	}

	return reply.DefaultLeaseTTL
}

func (s *SystemViewClient) MaxLeaseTTL() time.Duration {
	var reply MaxLeaseTTLReply
	err := s.client.Call("Plugin.MaxLeaseTTL", new(interface{}), &reply)
	if err != nil {
		return 0
	}

	return reply.MaxLeaseTTL
}

func (s *SystemViewClient) SudoPrivilege(ctx context.Context, path string, token string) bool {
	var reply SudoPrivilegeReply
	args := &SudoPrivilegeArgs{
		Path:  path,
		Token: token,
	}

	err := s.client.Call("Plugin.SudoPrivilege", args, &reply)
	if err != nil {
		return false
	}

	return reply.Sudo
}

func (s *SystemViewClient) Tainted() bool {
	var reply TaintedReply

	err := s.client.Call("Plugin.Tainted", new(interface{}), &reply)
	if err != nil {
		return false
	}

	return reply.Tainted
}

func (s *SystemViewClient) CachingDisabled() bool {
	var reply CachingDisabledReply

	err := s.client.Call("Plugin.CachingDisabled", new(interface{}), &reply)
	if err != nil {
		return false
	}

	return reply.CachingDisabled
}

func (s *SystemViewClient) ReplicationState() consts.ReplicationState {
	var reply ReplicationStateReply

	err := s.client.Call("Plugin.ReplicationState", new(interface{}), &reply)
	if err != nil {
		return consts.ReplicationUnknown
	}

	return reply.ReplicationState
}

func (s *SystemViewClient) ResponseWrapData(ctx context.Context, data map[string]interface{}, ttl time.Duration, jwt bool) (*wrapping.ResponseWrapInfo, error) {
	var reply ResponseWrapDataReply
	// Do not allow JWTs to be returned
	args := &ResponseWrapDataArgs{
		Data: data,
		TTL:  ttl,
		JWT:  false,
	}

	err := s.client.Call("Plugin.ResponseWrapData", args, &reply)
	if err != nil {
		return nil, err
	}
	if reply.Error != nil {
		return nil, reply.Error
	}

	return reply.ResponseWrapInfo, nil
}

func (s *SystemViewClient) LookupPlugin(_ context.Context, _ string, _ consts.PluginType) (*pluginutil.PluginRunner, error) {
	return nil, fmt.Errorf("cannot call LookupPlugin from a plugin backend")
}

func (s *SystemViewClient) HasFeature(feature license.Features) bool {
	// Not implemented
	return false
}

func (s *SystemViewClient) MlockEnabled() bool {
	var reply MlockEnabledReply
	err := s.client.Call("Plugin.MlockEnabled", new(interface{}), &reply)
	if err != nil {
		return false
	}

	return reply.MlockEnabled
}

func (s *SystemViewClient) LocalMount() bool {
	var reply LocalMountReply
	err := s.client.Call("Plugin.LocalMount", new(interface{}), &reply)
	if err != nil {
		return false
	}

	return reply.Local
}

func (s *SystemViewClient) EntityInfo(entityID string) (*logical.Entity, error) {
	var reply EntityInfoReply
	args := &EntityInfoArgs{
		EntityID: entityID,
	}

	err := s.client.Call("Plugin.EntityInfo", args, &reply)
	if err != nil {
		return nil, err
	}
	if reply.Error != nil {
		return nil, reply.Error
	}

	return reply.Entity, nil
}

func (s *SystemViewClient) PluginEnv(_ context.Context) (*logical.PluginEnvironment, error) {
	var reply PluginEnvReply

	err := s.client.Call("Plugin.PluginEnv", new(interface{}), &reply)
	if err != nil {
		return nil, err
	}
	if reply.Error != nil {
		return nil, reply.Error
	}

	return reply.PluginEnvironment, nil
}

type SystemViewServer struct {
	impl logical.SystemView
}

func (s *SystemViewServer) DefaultLeaseTTL(_ interface{}, reply *DefaultLeaseTTLReply) error {
	ttl := s.impl.DefaultLeaseTTL()
	*reply = DefaultLeaseTTLReply{
		DefaultLeaseTTL: ttl,
	}

	return nil
}

func (s *SystemViewServer) MaxLeaseTTL(_ interface{}, reply *MaxLeaseTTLReply) error {
	ttl := s.impl.MaxLeaseTTL()
	*reply = MaxLeaseTTLReply{
		MaxLeaseTTL: ttl,
	}

	return nil
}

func (s *SystemViewServer) SudoPrivilege(args *SudoPrivilegeArgs, reply *SudoPrivilegeReply) error {
	sudo := s.impl.SudoPrivilege(context.Background(), args.Path, args.Token)
	*reply = SudoPrivilegeReply{
		Sudo: sudo,
	}

	return nil
}

func (s *SystemViewServer) Tainted(_ interface{}, reply *TaintedReply) error {
	tainted := s.impl.Tainted()
	*reply = TaintedReply{
		Tainted: tainted,
	}

	return nil
}

func (s *SystemViewServer) CachingDisabled(_ interface{}, reply *CachingDisabledReply) error {
	cachingDisabled := s.impl.CachingDisabled()
	*reply = CachingDisabledReply{
		CachingDisabled: cachingDisabled,
	}

	return nil
}

func (s *SystemViewServer) ReplicationState(_ interface{}, reply *ReplicationStateReply) error {
	replicationState := s.impl.ReplicationState()
	*reply = ReplicationStateReply{
		ReplicationState: replicationState,
	}

	return nil
}

func (s *SystemViewServer) ResponseWrapData(args *ResponseWrapDataArgs, reply *ResponseWrapDataReply) error {
	// Do not allow JWTs to be returned
	info, err := s.impl.ResponseWrapData(context.Background(), args.Data, args.TTL, false)
	if err != nil {
		*reply = ResponseWrapDataReply{
			Error: wrapError(err),
		}
		return nil
	}
	*reply = ResponseWrapDataReply{
		ResponseWrapInfo: info,
	}

	return nil
}

func (s *SystemViewServer) MlockEnabled(_ interface{}, reply *MlockEnabledReply) error {
	enabled := s.impl.MlockEnabled()
	*reply = MlockEnabledReply{
		MlockEnabled: enabled,
	}

	return nil
}

func (s *SystemViewServer) LocalMount(_ interface{}, reply *LocalMountReply) error {
	local := s.impl.LocalMount()
	*reply = LocalMountReply{
		Local: local,
	}

	return nil
}

func (s *SystemViewServer) EntityInfo(args *EntityInfoArgs, reply *EntityInfoReply) error {
	entity, err := s.impl.EntityInfo(args.EntityID)
	if err != nil {
		*reply = EntityInfoReply{
			Error: wrapError(err),
		}
		return nil
	}
	*reply = EntityInfoReply{
		Entity: entity,
	}

	return nil
}

func (s *SystemViewServer) PluginEnv(_ interface{}, reply *PluginEnvReply) error {
	pluginEnv, err := s.impl.PluginEnv(context.Background())
	if err != nil {
		*reply = PluginEnvReply{
			Error: wrapError(err),
		}
		return nil
	}
	*reply = PluginEnvReply{
		PluginEnvironment: pluginEnv,
	}

	return nil
}

type DefaultLeaseTTLReply struct {
	DefaultLeaseTTL time.Duration
}

type MaxLeaseTTLReply struct {
	MaxLeaseTTL time.Duration
}

type SudoPrivilegeArgs struct {
	Path  string
	Token string
}

type SudoPrivilegeReply struct {
	Sudo bool
}

type TaintedReply struct {
	Tainted bool
}

type CachingDisabledReply struct {
	CachingDisabled bool
}

type ReplicationStateReply struct {
	ReplicationState consts.ReplicationState
}

type ResponseWrapDataArgs struct {
	Data map[string]interface{}
	TTL  time.Duration
	JWT  bool
}

type ResponseWrapDataReply struct {
	ResponseWrapInfo *wrapping.ResponseWrapInfo
	Error            error
}

type MlockEnabledReply struct {
	MlockEnabled bool
}

type LocalMountReply struct {
	Local bool
}

type EntityInfoArgs struct {
	EntityID string
}

type EntityInfoReply struct {
	Entity *logical.Entity
	Error  error
}

type PluginEnvReply struct {
	PluginEnvironment *logical.PluginEnvironment
	Error             error
}
