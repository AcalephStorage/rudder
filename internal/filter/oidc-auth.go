package filter

import (
	"strings"
	"time"

	"encoding/base64"
	"encoding/json"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-oidc"
	"github.com/emicklei/go-restful"
	"golang.org/x/net/context"
	"gopkg.in/square/go-jose.v2"

	"github.com/AcalephStorage/rudder/internal/util"
)

// OIDCAuth provides JWT verification
type OIDCAuth struct {
	issuerURL           string
	clientID            string
	clientSecret        string
	secretBase64Encoded bool
	verifier            *oidc.IDTokenVerifier
}

type idToken struct {
	Issuer   string   `json:"iss"`
	Subject  string   `json:"sub"`
	Audience audience `json:"aud"`
	Expiry   jsonTime `json:"exp"`
	IssuedAt jsonTime `json:"iat"`
	Nonce    string   `json:"nonce"`
}

// NewOIDCAuth returns a new OIDC authenticator. Token verification depends on the provided arguments.
// If oidcIssuer is provided and contains auto-discovery, RSA256 signed tokens can be verified. Providing
// clientSecret will also allow HS256 signed token verification.
//
// Other verifications are also done besides the signature. If oidcIssuer is provided, the iss claims will
// be verified. Providing clientID will verify the aud claim. Token expiry will always be verified.
func NewOIDCAuth(oidcIssuer, clientID, clientSecret string, secretBase64Encoded bool) *OIDCAuth {
	var verifier *oidc.IDTokenVerifier
	if len(oidcIssuer) > 0 {
		provider, err := oidc.NewProvider(context.Background(), oidcIssuer)
		if err != nil {
			log.WithError(err).Warn("Unable to connect to oidc issuer: Will not be able to verify RSA signed tokens")
		} else {
			options := []oidc.VerificationOption{oidc.VerifyExpiry()}
			if len(clientID) > 0 {
				options = append(options, oidc.VerifyAudience(clientID))
			}
			verifier = provider.Verifier(options...)
		}

	}
	return &OIDCAuth{
		issuerURL:           oidcIssuer,
		clientID:            clientID,
		clientSecret:        clientSecret,
		secretBase64Encoded: secretBase64Encoded,
		verifier:            verifier,
	}
}

// Authorize returns true if the req contains a valid token
func (oa *OIDCAuth) Authorize(req *restful.Request) (authorized bool) {
	log.Debug("verifying oidc token")
	authHeader := req.HeaderParameter("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		log.Debug("no bearer token provided")
		return false
	}
	rawIDToken := strings.Replace(authHeader, "Bearer ", "", -1)
	jws, err := jose.ParseSigned(rawIDToken)
	if err != nil {
		log.Debug("invalid token")
		return false
	}
	// go-oidc is convenient.. for rsa. so gotta do manual stuff here
	authorized = oa.verifyHS256(rawIDToken, jws)
	if !authorized {
		authorized = oa.verifyOthers(rawIDToken)
	}
	return
}

func (oa *OIDCAuth) verifyOthers(rawIDToken string) bool {
	if oa.verifier == nil {
		return false
	}
	_, err := oa.verifier.Verify(context.Background(), rawIDToken)
	if err != nil {
		log.WithError(err).Debug("unable to verify oidc token")
		return false
	}
	return true
}

// go-oidc doesn't provide convenience method for verifying HS256 but the library contains the means
// to do it but it's quite manual. This function does that.
func (oa *OIDCAuth) verifyHS256(rawIDToken string, jws *jose.JSONWebSignature) bool {
	if len(jws.Signatures) != 1 || !strings.HasPrefix(jws.Signatures[0].Header.Algorithm, "HS256") {
		// invalid or not HMAC signed
		return false
	}
	// verify signature
	var secret []byte
	if oa.secretBase64Encoded {
		secret, _ = base64.URLEncoding.DecodeString(oa.clientSecret)
	} else {
		secret = []byte(oa.clientSecret)
	}
	payload, err := jws.Verify(secret)
	if err != nil {
		log.WithError(err).Debug("failed to verify signature")
		return false
	}
	var token idToken
	if err := json.Unmarshal(payload, &token); err != nil {
		log.WithError(err).Debug("failed to read payload")
		return false
	}
	// verify issuer
	if oa.issuerURL != "" && token.Issuer != oa.issuerURL {
		log.Debug("invalid token issuer")
		return false
	}
	// verify audience
	if oa.clientID != "" {
		var found bool
		for _, aud := range token.Audience {
			if oa.clientID == aud {
				found = true
			}
		}
		if !found {
			log.Debug("invalid audience")
			return false
		}
	}
	// verify Expiry
	if util.IsExpired(time.Time(token.Expiry)) {
		log.Debug("token is expired")
		return false
	}
	return true
}

// json unmarshal helper

type audience []string

func (a *audience) UnmarshalJSON(b []byte) error {
	var s string
	if json.Unmarshal(b, &s) == nil {
		*a = audience{s}
		return nil
	}
	var auds []string
	if err := json.Unmarshal(b, &auds); err != nil {
		return err
	}
	*a = audience(auds)
	return nil
}

func (a audience) MarshalJSON() ([]byte, error) {
	if len(a) == 1 {
		return json.Marshal(a[0])
	}
	return json.Marshal([]string(a))
}

type jsonTime time.Time

func (j *jsonTime) UnmarshalJSON(b []byte) error {
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	var unix int64

	if t, err := n.Int64(); err == nil {
		unix = t
	} else {
		f, err := n.Float64()
		if err != nil {
			return err
		}
		unix = int64(f)
	}
	*j = jsonTime(time.Unix(unix, 0))
	return nil
}

func (j jsonTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(j).Unix())
}
