package types

import "github.com/yannick2009/yily/internal/id"

// Action identifies what happened in an audit event.
type Action string

const (
	ActionCreated   Action = "created"    // ActionCreated records the creation of a resource.
	ActionUpdated   Action = "updated"    // ActionUpdated records a change to a resource (rename, value, tags, …).
	ActionDeleted   Action = "deleted"    // ActionDeleted records the deletion of a resource.
	ActionRevealed  Action = "revealed"   // ActionRevealed records a decrypted secret value being returned.
	ActionExport    Action = "export"     // ActionExport records an environment's variables being exported.
	ActionImport    Action = "import"     // ActionImport records a file being imported into an environment.
	ActionRevoked   Action = "revoked"    // ActionRevoked records the revocation of a Personal Access Token.
	ActionRotated   Action = "rotated"    // ActionRotated records the rotation of a Personal Access Token (old revoked, new issued under the same name).
	ActionSignedIn  Action = "signed_in"  // ActionSignedIn records a successful login.
	ActionSignedOut Action = "signed_out" // ActionSignedOut records an explicit logout.
)

// ActorType identifies who performed an audited action.
type ActorType string

const (
	ActorTypeUser   ActorType = "user"   // ActorTypeUser is a super admin acting through a session.
	ActorTypeToken  ActorType = "token"  // ActorTypeToken is a Personal Access Token acting through the API.
	ActorTypeSystem ActorType = "system" // ActorTypeSystem is an internal/automatic action without a user.
)

// AuditActor identifies who performed an audited action.
type AuditActor struct {
	Type ActorType // Type is the kind of actor (user, token or system).
	ID   id.ID     // ID is the actor's identifier ("" for "system").
	Name string    // Name is the actor's display name (empty for "system").
}

// ResourceType identifies the kind of resource an audit event concerns.
type ResourceType string

const (
	ResourceProject     ResourceType = "project"     // ResourceProject is a project.
	ResourceEnvironment ResourceType = "environment" // ResourceEnvironment is an environment.
	ResourceVariable    ResourceType = "variable"    // ResourceVariable is a variable (secret).
	ResourceTag         ResourceType = "tag"         // ResourceTag is a tag.
	ResourceToken       ResourceType = "token"       // ResourceToken is a Personal Access Token.
)
