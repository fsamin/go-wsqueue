package wsqueue

import "net/http"

//ACL  stands for Access Control List. It's a slice of permission for a queue or a topic
type ACL []ACE

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

func checkACL(acl ACL, w http.ResponseWriter, r *http.Request) bool {
	for _, ace := range acl {
		switch ace.Scheme() {
		case ACLSSchemeWorld:
			Logfunc("Connection Authorized")
			return true
		case ACLSSchemeIP:
			ip := r.Header.Get("X-Forwarded-For")
			aceIP, b := ace.(*ACEIP)
			if !b {
				w.WriteHeader(http.StatusUnauthorized)
				return false
			}
			if ip == aceIP.IP {
				Logfunc("Connection Authorized for IP %s", ip)
				return true
			}
			Warnfunc("Connection unauthorized for IP:%s", ip)
		case ACLSSchemeDigest:
			u, p, b := r.BasicAuth()
			if b {
				aceDigest, b := ace.(*ACEDigest)
				if b && aceDigest.Username == u && aceDigest.Password == p {
					Logfunc("Connection Authorized with BasicAuth %s", u)
					return true
				}
			}
			Warnfunc("Connection unauthorized for BasicAuth %s", u)
		}
	}
	w.WriteHeader(http.StatusUnauthorized)
	return false
}
