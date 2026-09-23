package otp

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Entry struct {
	Secret    string    `json:"secret"`
	Algorithm string    `json:"algorithm"`
	Digits    int       `json:"digits"`
	Period    int       `json:"period"`
	CreatedAt time.Time `json:"created_at"`
}

var algorithms = map[string]func() hash.Hash{
	"SHA1":   sha1.New,
	"SHA256": sha256.New,
	"SHA512": sha512.New,
}

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

func Parse(input string) (Entry, error) {
	input = strings.TrimSpace(input)
	e := Entry{Algorithm: "SHA1", Digits: 6, Period: 30}
	if !strings.HasPrefix(strings.ToLower(input), "otpauth://") {
		e.Secret = input
		return e, e.normalize()
	}

	u, err := url.Parse(input)
	if err != nil {
		return Entry{}, err
	}
	if !strings.EqualFold(u.Host, "totp") {
		return Entry{}, fmt.Errorf("unsupported otpauth type %q, only totp is supported", u.Host)
	}
	q := u.Query()
	e.Secret = q.Get("secret")
	if a := q.Get("algorithm"); a != "" {
		e.Algorithm = a
	}
	for key, dst := range map[string]*int{"digits": &e.Digits, "period": &e.Period} {
		if v := q.Get(key); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil {
				return Entry{}, fmt.Errorf("invalid %s %q", key, v)
			}
			*dst = n
		}
	}
	return e, e.normalize()
}

func (e *Entry) normalize() error {
	e.Secret = strings.TrimRight(strings.ToUpper(strings.NewReplacer(" ", "", "-", "").Replace(e.Secret)), "=")
	if e.Secret == "" {
		return errors.New("secret is empty")
	}
	switch len(e.Secret) % 8 {
	case 1, 3, 6:
		return fmt.Errorf("secret has invalid base32 length %d", len(e.Secret))
	}
	if _, err := b32.DecodeString(e.Secret); err != nil {
		return fmt.Errorf("secret is not valid base32: %w", err)
	}
	e.Algorithm = strings.ToUpper(e.Algorithm)
	if _, ok := algorithms[e.Algorithm]; !ok {
		return fmt.Errorf("unsupported algorithm %q", e.Algorithm)
	}
	if e.Digits < 6 || e.Digits > 8 {
		return fmt.Errorf("digits must be between 6 and 8, got %d", e.Digits)
	}
	if e.Period <= 0 {
		return fmt.Errorf("period must be positive, got %d", e.Period)
	}
	return nil
}

func (e Entry) Code(t time.Time) (string, error) {
	key, err := b32.DecodeString(e.Secret)
	if err != nil {
		return "", err
	}
	newHash, ok := algorithms[e.Algorithm]
	if !ok {
		return "", fmt.Errorf("unsupported algorithm %q", e.Algorithm)
	}
	counter := make([]byte, 8)
	binary.BigEndian.PutUint64(counter, uint64(t.Unix())/uint64(e.Period))
	mac := hmac.New(newHash, key)
	mac.Write(counter)
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	bin := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	mod := uint32(1)
	for range e.Digits {
		mod *= 10
	}
	return fmt.Sprintf("%0*d", e.Digits, bin%mod), nil
}
