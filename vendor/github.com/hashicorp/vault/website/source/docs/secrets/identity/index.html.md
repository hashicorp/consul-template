---
layout: "docs"
page_title: "Identity - Secrets Engines"
sidebar_title: "Identity"
sidebar_current: "docs-secrets-identity"
description: |-
  The Identity secrets engine for Vault manages client identities.
---

# Identity Secrets Engine

Name: `identity`

The Identity secrets engine is the identity management solution for Vault. It
internally maintains the clients who are recognized by Vault. Each client is
internally termed as an `Entity`. An entity can have multiple `Aliases`. For
example, a single user who has accounts in both GitHub and LDAP, can be mapped
to a single entity in Vault that has 2 aliases, one of type GitHub and one of
type LDAP. When a client authenticates via any of the credential backend
(except the Token backend), Vault creates a new entity and attaches a new
alias to it, if a corresponding entity doesn't already exist. The entity identifier will
be tied to the authenticated token. When such tokens are put to use, their
entity identifiers are audit logged, marking a trail of actions performed by
specific users.

Identity store allows operators to **manage** the entities in Vault. Entities
can be created and aliases can be tied to entities, via the ACL'd API. There
can be policies set on the entities which adds capabilities to the tokens that
are tied to entity identifiers. The capabilities granted to tokens via the
entities are **an addition** to the existing capabilities of the token and
**not** a replacement. The capabilities of the token that get inherited from
entities are computed dynamically at request time. This provides flexibility in
controlling the access of tokens that are already issued.

This secrets engine will be mounted by default. This secrets engine cannot be
disabled or moved.

## Concepts

### Entities and Aliases

Each user will have multiple accounts with various identity providers. Users
can now be mapped as `Entities` and their corresponding accounts with
authentication providers can be mapped as `Aliases`. In essence, each entity is
made up of zero or more aliases.

### Entity Management

Entities in Vault **do not** automatically pull identity information from
anywhere. It needs to be explicitly managed by operators. This way, it is
flexible in terms of administratively controlling the number of entities to be
synced against Vault. In some sense, Vault will serve as a _cache_ of
identities and not as a _source_ of identities.

### Entity Policies

Vault policies can be assigned to entities which will grant _additional_
permissions to the token on top of the existing policies on the token. If the
token presented on the API request contains an identifier for the entity and if
that entity has a set of policies on it, then the token will be capable of
performing actions allowed by the policies on the entity as well.

This is a paradigm shift in terms of _when_ the policies of the token get
evaluated. Before identity, the policy names on the token were immutable (not
the contents of those policies though). But with entity policies, along with
the immutable set of policy names on the token, the evaluation of policies
applicable to the token through its identity will happen at request time. This
also adds enormous flexibility to control the behavior of already issued
tokens.

Its important to note that the policies on the entity are only a means to grant
_additional_ capabilities and not a replacement for the policies on the token.
To know the full set of capabilities of the token with an associated entity
identifier, the policies on the token should be taken into account.

### Mount Bound Aliases

Vault supports multiple authentication backends and also allows enabling the
same type of authentication backend on different mount paths. The alias name of
the user will be unique within the backend's mount. But identity store needs to
uniquely distinguish between conflicting alias names across different mounts of
these identity providers. Hence, the alias name in combination with the
authentication backend mount's accessor, serve as the unique identifier of an
alias.

### Implicit Entities

Operators can create entities for all the users of an auth mount beforehand and
assign policies to them, so that when users login, the desired capabilities to
the tokens via entities are already assigned. But if that's not done, upon a
successful user login from any of the authentication backends, Vault will
create a new entity and assign an alias against the login that was successful.

Note that the tokens created using the token authentication backend will not
have an associated identity information. Logging in using the authentication
backends is the only way to create tokens that have a valid entity identifiers.

### Identity Auditing

If the token used to make API calls have an associated entity identifier, it
will be audit logged as well. This leaves a trail of actions performed by
specific users.

### Identity Groups

In version 0.9, Vault identity has support for groups. A group can contain
multiple entities as its members. A group can also have subgroups. Policies set
on the group is granted to all members of the group. During request time, when
the token's entity ID is being evaluated for the policies that it has access
to; along with the policies on the entity itself, policies that are inherited
due to group memberships are also granted.

### Group Hierarchical Permissions

Entities can be direct members of groups, in which case they inherit the
policies of the groups they belong to. Entities can also be indirect members of
groups. For example, if a GroupA has GroupB as subgroup, then members of GroupB
are indirect members of GroupA. Hence, the members of GroupB will have access
to policies on both GroupA and GroupB.

### External vs Internal Groups

By default, the groups created in identity store are called the internal
groups. The membership management of these groups should be carried out
manually. A group can also be created as an external group. In this case, the
entity membership in the group is managed semi-automatically. External group
serves as a mapping to a group that is outside of the identity store. External
groups can have one (and only one) alias. This alias should map to a notion of
group that is outside of the identity store. For example, groups in LDAP, and
teams in GitHub. A username in LDAP, belonging to a group in LDAP, can get its
entity ID added as a member of a group in Vault automatically during *logins*
and *token renewals*. This works only if the group in Vault is an external
group and has an alias that maps to the group in LDAP. If the user is removed
from the group in LDAP, that change gets reflected in Vault only upon the
subsequent login or renewal operation.



## API

The Identity secrets engine has a full HTTP API. Please see the
[Identity secrets engine API](/api/secret/identity/index.html) for more
details.
