package vault

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/vault/helper/identity"
	"github.com/hashicorp/vault/helper/namespace"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

func groupAliasPaths(i *IdentityStore) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "group-alias$",
			Fields: map[string]*framework.FieldSchema{
				"id": {
					Type:        framework.TypeString,
					Description: "ID of the group alias.",
				},
				"name": {
					Type:        framework.TypeString,
					Description: "Alias of the group.",
				},
				"mount_accessor": {
					Type:        framework.TypeString,
					Description: "Mount accessor to which this alias belongs to.",
				},
				"canonical_id": {
					Type:        framework.TypeString,
					Description: "ID of the group to which this is an alias.",
				},
			},
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.UpdateOperation: i.pathGroupAliasRegister(),
			},

			HelpSynopsis:    strings.TrimSpace(groupAliasHelp["group-alias"][0]),
			HelpDescription: strings.TrimSpace(groupAliasHelp["group-alias"][1]),
		},
		{
			Pattern: "group-alias/id/" + framework.GenericNameRegex("id"),
			Fields: map[string]*framework.FieldSchema{
				"id": {
					Type:        framework.TypeString,
					Description: "ID of the group alias.",
				},
				"name": {
					Type:        framework.TypeString,
					Description: "Alias of the group.",
				},
				"mount_accessor": {
					Type:        framework.TypeString,
					Description: "Mount accessor to which this alias belongs to.",
				},
				"canonical_id": {
					Type:        framework.TypeString,
					Description: "ID of the group to which this is an alias.",
				},
			},
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.UpdateOperation: i.pathGroupAliasIDUpdate(),
				logical.ReadOperation:   i.pathGroupAliasIDRead(),
				logical.DeleteOperation: i.pathGroupAliasIDDelete(),
			},

			HelpSynopsis:    strings.TrimSpace(groupAliasHelp["group-alias-by-id"][0]),
			HelpDescription: strings.TrimSpace(groupAliasHelp["group-alias-by-id"][1]),
		},
		{
			Pattern: "group-alias/id/?$",
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.ListOperation: i.pathGroupAliasIDList(),
			},

			HelpSynopsis:    strings.TrimSpace(groupAliasHelp["group-alias-id-list"][0]),
			HelpDescription: strings.TrimSpace(groupAliasHelp["group-alias-id-list"][1]),
		},
	}
}

func (i *IdentityStore) pathGroupAliasRegister() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
		_, ok := d.GetOk("id")
		if ok {
			return i.pathGroupAliasIDUpdate()(ctx, req, d)
		}

		i.groupLock.Lock()
		defer i.groupLock.Unlock()

		return i.handleGroupAliasUpdateCommon(ctx, req, d, nil)
	}
}

func (i *IdentityStore) pathGroupAliasIDUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
		groupAliasID := d.Get("id").(string)
		if groupAliasID == "" {
			return logical.ErrorResponse("empty group alias ID"), nil
		}

		i.groupLock.Lock()
		defer i.groupLock.Unlock()

		groupAlias, err := i.MemDBAliasByID(groupAliasID, true, true)
		if err != nil {
			return nil, err
		}
		if groupAlias == nil {
			return logical.ErrorResponse("invalid group alias ID"), nil
		}

		return i.handleGroupAliasUpdateCommon(ctx, req, d, groupAlias)
	}
}

func (i *IdentityStore) handleGroupAliasUpdateCommon(ctx context.Context, req *logical.Request, d *framework.FieldData, groupAlias *identity.Alias) (*logical.Response, error) {
	var newGroupAlias bool
	var group *identity.Group
	var err error

	if groupAlias == nil {
		groupAlias = &identity.Alias{}
		newGroupAlias = true
	}

	groupID := d.Get("canonical_id").(string)
	if groupID != "" {
		group, err = i.MemDBGroupByID(groupID, true)
		if err != nil {
			return nil, err
		}
		if group == nil {
			return logical.ErrorResponse("invalid group ID"), nil
		}
		if group.Type != groupTypeExternal {
			return logical.ErrorResponse("alias can't be set on an internal group"), nil
		}
	}

	// Get group alias name
	groupAliasName := d.Get("name").(string)
	if groupAliasName == "" {
		return logical.ErrorResponse("missing alias name"), nil
	}

	mountAccessor := d.Get("mount_accessor").(string)
	if mountAccessor == "" {
		return logical.ErrorResponse("missing mount_accessor"), nil
	}

	mountValidationResp := i.core.router.validateMountByAccessor(mountAccessor)
	if mountValidationResp == nil {
		return logical.ErrorResponse(fmt.Sprintf("invalid mount accessor %q", mountAccessor)), nil
	}

	if mountValidationResp.MountLocal {
		return logical.ErrorResponse(fmt.Sprintf("mount_accessor %q is of a local mount", mountAccessor)), nil
	}

	groupAliasByFactors, err := i.MemDBAliasByFactors(mountValidationResp.MountAccessor, groupAliasName, false, true)
	if err != nil {
		return nil, err
	}

	resp := &logical.Response{}

	if newGroupAlias {
		if groupAliasByFactors != nil {
			return logical.ErrorResponse("combination of mount and group alias name is already in use"), nil
		}

		// If this is an alias being tied to a non-existent group, create
		// a new group for it.
		if group == nil {
			group = &identity.Group{
				Type:  groupTypeExternal,
				Alias: groupAlias,
			}
		} else {
			group.Alias = groupAlias
		}
	} else {
		// Verify that the combination of group alias name and mount is not
		// already tied to a different alias
		if groupAliasByFactors != nil && groupAliasByFactors.ID != groupAlias.ID {
			return logical.ErrorResponse("combination of mount and group alias name is already in use"), nil
		}

		// Fetch the group to which the alias is tied to
		existingGroup, err := i.MemDBGroupByAliasID(groupAlias.ID, true)
		if err != nil {
			return nil, err
		}

		if existingGroup == nil {
			return nil, fmt.Errorf("group alias is not associated with a group")
		}

		if group != nil && group.ID != existingGroup.ID {
			return logical.ErrorResponse("alias is already tied to a different group"), nil
		}

		group = existingGroup
		group.Alias = groupAlias
	}

	group.Alias.Name = groupAliasName
	group.Alias.MountAccessor = mountValidationResp.MountAccessor
	// Explicitly correct for previous versions that persisted this
	group.Alias.MountType = ""

	err = i.sanitizeAndUpsertGroup(ctx, group, nil)
	if err != nil {
		return nil, err
	}

	resp.Data = map[string]interface{}{
		"id":           groupAlias.ID,
		"canonical_id": group.ID,
	}

	return resp, nil
}

// pathGroupAliasIDRead returns the properties of an alias for a given
// alias ID
func (i *IdentityStore) pathGroupAliasIDRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
		groupAliasID := d.Get("id").(string)
		if groupAliasID == "" {
			return logical.ErrorResponse("empty group alias id"), nil
		}

		groupAlias, err := i.MemDBAliasByID(groupAliasID, false, true)
		if err != nil {
			return nil, err
		}

		return i.handleAliasReadCommon(ctx, groupAlias)
	}
}

// pathGroupAliasIDDelete deletes the group's alias for a given group alias ID
func (i *IdentityStore) pathGroupAliasIDDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
		groupAliasID := d.Get("id").(string)
		if groupAliasID == "" {
			return logical.ErrorResponse("missing group alias ID"), nil
		}

		i.groupLock.Lock()
		defer i.groupLock.Unlock()

		txn := i.db.Txn(true)
		defer txn.Abort()

		alias, err := i.MemDBAliasByIDInTxn(txn, groupAliasID, false, true)
		if err != nil {
			return nil, err
		}

		if alias == nil {
			return nil, nil
		}

		ns, err := namespace.FromContext(ctx)
		if err != nil {
			return nil, err
		}
		if ns.ID != alias.NamespaceID {
			return nil, logical.ErrUnsupportedOperation
		}

		group, err := i.MemDBGroupByAliasIDInTxn(txn, alias.ID, true)
		if err != nil {
			return nil, err
		}

		// If there is no group tied to a valid alias, something is wrong
		if group == nil {
			return nil, fmt.Errorf("alias not associated to a group")
		}

		// Delete group alias in memdb
		err = i.MemDBDeleteAliasByIDInTxn(txn, group.Alias.ID, true)
		if err != nil {
			return nil, err
		}

		// Delete the alias
		group.Alias = nil

		err = i.UpsertGroupInTxn(txn, group, true)
		if err != nil {
			return nil, err
		}

		txn.Commit()

		return nil, nil
	}
}

// pathGroupAliasIDList lists the IDs of all the valid group aliases in the
// identity store
func (i *IdentityStore) pathGroupAliasIDList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
		return i.handleAliasListCommon(ctx, true)
	}
}

var groupAliasHelp = map[string][2]string{
	"group-alias": {
		"Creates a new group alias, or updates an existing one.",
		"",
	},
	"group-alias-id": {
		"Update, read or delete a group alias using ID.",
		"",
	},
	"group-alias-id-list": {
		"List all the group alias IDs.",
		"",
	},
}
