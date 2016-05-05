package wsqueue

type QueueACL []QueueACE

type QueueACE struct {
	Scheme QueueACLScheme `json:"scheme,omitempty"`
	Perm   QueueACLPerm   `json:"perm,omitempty"`
}

type QueueACEDigest struct {
	QueueACE
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type QueueACEIP struct {
	QueueACE
	IP string `json:"ip,omitempty"`
}

type QueueACLScheme string

const (
	QUEUE_ACL_SCHEME_WORLD  QueueACLScheme = "WORLD"
	QUEUE_ACL_SCHEME_DIGEST                = "DIGEST"
	QUEUE_ACL_SCHEME_IP                    = "IP"
)

type QueueACLPerm string

const (
	QUEUE_ACL_PERM_READ  QueueACLPerm = "READ"
	QUEUE_ACL_PERM_WRITE              = "WRITE"
	QUEUE_ACL_PERM_ALL                = "ALL"
	QUEUE_ACL_PERM_ADMIN              = "ADMIN"
)
