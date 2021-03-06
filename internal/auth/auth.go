package auth

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"blitiri.com.ar/go/chasquid/internal/normalize"
	"blitiri.com.ar/go/chasquid/internal/userdb"
)

// DecodeResponse decodes a plain auth response.
//
// It must be a a base64-encoded string of the form:
//   <authorization id> NUL <authentication id> NUL <password>
//
// https://tools.ietf.org/html/rfc4954#section-4.1.
//
// Either both ID match, or one of them is empty.
// We expect the ID to be "user@domain", which is NOT an RFC requirement but
// our own.
func DecodeResponse(response string) (user, domain, passwd string, err error) {
	buf, err := base64.StdEncoding.DecodeString(response)
	if err != nil {
		return
	}

	bufsp := bytes.SplitN(buf, []byte{0}, 3)
	if len(bufsp) != 3 {
		err = fmt.Errorf("Response pieces != 3, as per RFC")
		return
	}

	identity := ""
	passwd = string(bufsp[2])

	{
		// We don't make the distinction between the two IDs, as long as one is
		// empty, or they're the same.
		z := string(bufsp[0])
		c := string(bufsp[1])

		// If neither is empty, then they must be the same.
		if (z != "" && c != "") && (z != c) {
			err = fmt.Errorf("Auth IDs do not match")
			return
		}

		if z != "" {
			identity = z
		}
		if c != "" {
			identity = c
		}
	}

	if identity == "" {
		err = fmt.Errorf("Empty identity, must be in the form user@domain")
		return
	}

	// Identity must be in the form "user@domain".
	// This is NOT an RFC requirement, it's our own.
	idsp := strings.SplitN(identity, "@", 2)
	if len(idsp) != 2 {
		err = fmt.Errorf("Identity must be in the form user@domain")
		return
	}

	user = idsp[0]
	domain = idsp[1]

	// Normalize the user and domain. This is so users can write the username
	// in their own style and still can log in.  For the domain, we use IDNA
	// and relevant transformations to turn it to utf8 which is what we use
	// internally.
	user, err = normalize.User(user)
	if err != nil {
		return
	}
	domain, err = normalize.Domain(domain)
	if err != nil {
		return
	}

	return
}

// How long Authenticate calls should last, approximately.
// This will be applied both for successful and unsuccessful attempts.
// We will increase this number by 0-20%.
var AuthenticateTime = 100 * time.Millisecond

// Authenticate user/password on the given database.
func Authenticate(udb *userdb.DB, user, passwd string) bool {
	defer func(start time.Time) {
		elapsed := time.Since(start)
		delay := AuthenticateTime - elapsed
		if delay > 0 {
			maxDelta := int64(float64(delay) * 0.2)
			delay += time.Duration(rand.Int63n(maxDelta))
			time.Sleep(delay)
		}
	}(time.Now())

	// Note that the database CAN be nil, to simplify callers.
	if udb == nil {
		return false
	}

	return udb.Authenticate(user, passwd)
}
