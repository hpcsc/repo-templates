package idempotency

import (
	"net/http"

	"github.com/google/uuid"
)

const Header = "Idempotency-Key"

func IDFor(req *http.Request, purpose string) uuid.UUID {
	key := req.Header.Get(Header)
	if key == "" {
		return uuid.New()
	}
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(purpose+" "+req.Method+" "+req.URL.Path+" "+key))
}
