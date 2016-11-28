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
)

type OIDCAuth struct {
	OIDCIssuer          string
	ClientID            string
	ClientSecret        string
	SecretBase64Encoded bool
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

func NewOIDCAuth(oidcIssuer, clientID, clientSecret string, secretBase64Encoded bool) (*OIDCAuth, error) {

	var verifier *oidc.IDTokenVerifier
	if len(oidcIssuer) > 0 {
		provider, err := oidc.NewProvider(context.Background(), oidcIssuer)
		if err != nil {
			log.WithError(err).Error("Unable to connect to oidc issuer")
			return nil, err
		}
		options := []oidc.VerificationOption{oidc.VerifyExpiry()}
		if len(clientID) > 0 {
			options = append(options, oidc.VerifyAudience(clientID))
		}
		verifier = provider.Verifier(options...)
	}

	return &OIDCAuth{
		OIDCIssuer:          oidcIssuer,
		ClientID:            clientID,
		ClientSecret:        clientSecret,
		SecretBase64Encoded: secretBase64Encoded,
		verifier:            verifier,
	}, nil
}

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

func (oa *OIDCAuth) verifyHS256(rawIDToken string, jws *jose.JSONWebSignature) bool {
	if len(jws.Signatures) != 1 || !strings.HasPrefix(jws.Signatures[0].Header.Algorithm, "HS256") {
		// invalid or not HMAC
		return false
	}
	// verify signature
	var secret []byte
	if oa.SecretBase64Encoded {
		secret, _ = base64.URLEncoding.DecodeString(oa.ClientSecret)
	} else {
		secret = []byte(oa.ClientSecret)
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
	if len(oa.OIDCIssuer) > 0 && token.Issuer != oa.OIDCIssuer {
		log.Debug("invalid token issuer")
		return false
	}
	// verify audience
	if len(oa.ClientID) > 0 {
		var found bool
		for _, aud := range token.Audience {
			if oa.ClientID == aud {
				found = true
			}
		}
		if !found {
			log.Debug("invalid audience")
			return false
		}
	}
	// verify Expiry
	if time.Now().After(time.Time(token.Expiry)) {
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
