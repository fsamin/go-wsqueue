package wsqueue

//ACL  stands for Access Control List. It's a slice of permission for a queue or a topic
type ACL struct {
	ListACEs []ACE `json:"list_ACE,omitempty"`
	queue    *Queue
	topic    *Topic
}

//ACE stands for Access Control Entity
type ACE interface {
	Scheme() ACLScheme
}

//ACEDigest aims to authenticate a user with a username and a password
type ACEDigest struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

//Scheme return WORLD, DIGEST or IP
func (a *ACEDigest) Scheme() ACLScheme {
	return ACLSSchemeDigest
}

//ACEIP aims to authenticate a user with a IP adress
type ACEIP struct {
	IP string `json:"ip,omitempty"`
}

//Scheme is IP
func (a *ACEIP) Scheme() ACLScheme {
	return ACLSSchemeIP
}

//ACEWorld -> everyone
type ACEWorld struct{}

//Scheme is World
func (a *ACEWorld) Scheme() ACLScheme {
	return ACLSSchemeWorld
}

//ACLScheme : There are three different scheme
type ACLScheme string

const (
	//ACLSSchemeWorld scheme is a fully open scheme
	ACLSSchemeWorld ACLScheme = "WORLD"
	//ACLSSchemeDigest scheme represents a "manually" set group of authenticated users.
	ACLSSchemeDigest = "DIGEST"
	//ACLSSchemeIP scheme represents a "manually" set group of  user authenticated by their IP address
	ACLSSchemeIP = "IP"
)
