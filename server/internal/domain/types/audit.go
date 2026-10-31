package types

// Action identifies what happened in an audit event.
type Action string

const (
	ActionCreated   Action = "created"    // ActionCreated records the creation of a resource.
	ActionUpdated   Action = "updated"    // ActionUpdated records a change to a resource (rename, value, tags, …).
	ActionDeleted   Action = "deleted"    // ActionDeleted records the deletion of a resource.
	ActionRevealed  Action = "revealed"   // ActionRevealed records a decrypted secret value being returned.
	ActionExport    Action = "export"     // ActionExport records an environment's variables being exported.
	ActionRevoked   Action = "revoked"    // ActionRevoked records the revocation of a Personal Access Token.
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

// ResourceType identifies the kind of resource an audit event concerns.
type ResourceType string

const (
	ResourceProject     ResourceType = "project"     // ResourceProject is a project.
	ResourceEnvironment ResourceType = "environment" // ResourceEnvironment is an environment.
	ResourceVariable    ResourceType = "variable"    // ResourceVariable is a variable (secret).
	ResourceTag         ResourceType = "tag"         // ResourceTag is a tag.
	ResourceToken       ResourceType = "token"       // ResourceToken is a Personal Access Token.
)
