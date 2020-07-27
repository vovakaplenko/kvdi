package v1

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// API Request/Response types

// LoginRequest represents a request for a session token. Different auth providers
// may not always need this request, and can instead redirect /api/login as needed.
// All the auth provider needs to do in the end is return a JWT token that contains
// a fulfilled VDIUser.
// +k8s:deepcopy-gen=false
type LoginRequest struct {
	// Username
	Username string `json:"username"`
	// Password
	Password string `json:"password"`
	// State generated by requesting client to prevent CSRF and retrieve tokens
	// from an oidc flow
	State string `json:"state"`
	// the underlying request object for usage by auth providers
	request *http.Request
}

// SetRequest sets the request object in the LoginRequest.
func (l *LoginRequest) SetRequest(r *http.Request) {
	l.request = r
}

// GetRequest returns the request object in the LoginRequest.
func (l *LoginRequest) GetRequest() *http.Request {
	return l.request
}

// AuthorizeRequest is a request with an OTP for receiving an authorized token.
type AuthorizeRequest struct {
	// The one-time password
	OTP string `json:"otp"`
}

// SessionResponse represents a response with a new session token
type SessionResponse struct {
	// The X-Session-Token to use for future requests.
	Token string `json:"token"`
	// The time the token expires.
	ExpiresAt int64 `json:"expiresAt"`
	// Information about the authenticated user and their permissions.
	User *VDIUser `json:"user"`
	// Whether the user is fully authorized (e.g. false if MFA is required but not provided yet)
	Authorized bool `json:"authorized"`
}

// CreateUserRequest represents a request to create a new user. Not all auth
// providers will be able to implement this route and can instead return an
// error describing why.
type CreateUserRequest struct {
	// The user name for the new user.
	Username string `json:"username"`
	// The password for the new user.
	Password string `json:"password"`
	// Roles to assign the new user. These are the names of VDIRoles in the cluster.
	Roles []string `json:"roles"`
}

// Validate validates a new user request
func (r *CreateUserRequest) Validate() error {
	if r.Username == "" || r.Password == "" {
		return errors.New("'username' and 'password' must be provided in the request")
	}
	if r.Roles == nil || len(r.Roles) == 0 {
		return errors.New("You must assign at least one role to the user")
	}
	if strings.Contains(r.Username, ":") {
		return errors.New("Username cannot contain the ':' character")
	}
	return nil
}

// UpdateUserRequest requests updates to an existing user. Not all auth
// providers will be able to implement this route and can instead return an
// error describing why.
type UpdateUserRequest struct {
	// When populated, will change the password for the user.
	Password string `json:"password"`
	// When populated will change the roles for the user.
	Roles []string `json:"roles"`
}

// Validate the UpdateUserRequest
func (r *UpdateUserRequest) Validate() error {
	if r.Password == "" && len(r.Roles) == 0 {
		return errors.New("You must specify either a new password or a list of roles")
	}
	return nil
}

// UpdateMFARequest sets the MFA configuration for the user. If enabling,
// a provisioning URI will be returned.
type UpdateMFARequest struct {
	// When set, will enable MFA for the given user. If false, will disable MFA.
	Enabled bool `json:"enabled"`
}

// MFAResponse contains the response to an UpdateMFARequest or GetMFARequest.
type MFAResponse struct {
	// Whether MFA is enabled for the user
	Enabled bool `json:"enabled"`
	// If enabled is set, a provisioning URI is also returned.
	ProvisioningURI string `json:"provisioningURI"`
	// If enabled is set, whether or not the user has verified their MFA setup
	Verified bool `json:"verified"`
}

// CreateRoleRequest represents a request for a new role.
type CreateRoleRequest struct {
	// The name of the new role
	Name string `json:"name"`
	// Annotations to apply to the role
	Annotations map[string]string `json:"annotations"`
	// Rules to apply to the new role.
	Rules []Rule `json:"rules"`
}

// GetName returns the name of the new role
func (r *CreateRoleRequest) GetName() string { return r.Name }

// GetAnnotations returns the annotations provided in the request
func (r *CreateRoleRequest) GetAnnotations() map[string]string { return r.Annotations }

// Validate the CreateRoleRequest
func (r *CreateRoleRequest) Validate() error {
	if r.Name == "" {
		return errors.New("A name is required for the new role")
	}
	for _, rule := range r.Rules {
		if err := validatePatterns(rule.ResourcePatterns); err != nil {
			return err
		}
	}
	return nil
}

// GetRules returns the rules for a new role request, or a single-element slice with
// a deny-all rule if none are provided.
func (r *CreateRoleRequest) GetRules() []Rule {
	if r.Rules == nil {
		return []Rule{{
			Verbs:            []Verb{},
			Resources:        []Resource{},
			ResourcePatterns: []string{},
			Namespaces:       []string{},
		}}
	}
	return r.Rules
}

// UpdateRoleRequest requests updates to an existing role. The existing attributes
// will be entirely replaced with those supplied in the payload.
type UpdateRoleRequest struct {
	// The new annotations for the role
	Annotations map[string]string `json:"annotations"`
	// The new rules for the role.
	Rules []Rule `json:"rules"`
}

// GetAnnotations returns the annotations provided in the request
func (r *UpdateRoleRequest) GetAnnotations() map[string]string { return r.Annotations }

// GetRules returns the rules for an update role request, or a single-element slice with
// a deny-all rule if none are provided.
func (r *UpdateRoleRequest) GetRules() []Rule {
	if r.Rules == nil {
		return []Rule{{
			Verbs:            []Verb{},
			Resources:        []Resource{},
			ResourcePatterns: []string{},
			Namespaces:       []string{},
		}}
	}
	return r.Rules
}

// Validate the UpdateRoleRequest
func (r *UpdateRoleRequest) Validate() error {
	for _, rule := range r.Rules {
		if err := validatePatterns(rule.ResourcePatterns); err != nil {
			return err
		}
	}
	return nil
}

// validatePatterns takes a list of regexes and returns an error if any of them
// are invalid.
func validatePatterns(patterns []string) error {
	for _, pattern := range patterns {
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("%s is an invalid regex: %s", pattern, err.Error())
		}
	}
	return nil
}

// CreateSessionRequest requests a new desktop session with the givin parameters.
type CreateSessionRequest struct {
	// The template to create the session from.
	Template string `json:"template"`
	// The namespace to launch the template in. Defaults to default.
	Namespace string `json:"namespace,omitempty"`
}

// Validate the CreateSessionRequest
func (r *CreateSessionRequest) Validate() error {
	if r.Template == "" {
		return errors.New("A template is required")
	}
	return nil
}

// GetTemplate returns the template for this request
func (r *CreateSessionRequest) GetTemplate() string {
	return r.Template
}

// GetNamespace returns the namspace for this request, or the default namespace
// if not provided.
func (r *CreateSessionRequest) GetNamespace() string {
	if r.Namespace != "" {
		return r.Namespace
	}
	return DefaultNamespace
}
