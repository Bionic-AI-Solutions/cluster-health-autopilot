// Copyright 2026 Cluster Health Autopilot contributors
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"crypto/ed25519"
	"fmt"
	"net/url"
	"time"
)

// SilenceLinkRequest describes a finding for which one-click silence
// links should be minted.
type SilenceLinkRequest struct {
	// Source is the finding's analyzer name → matcher.source on both
	// the subject- and class-scoped silences. Required.
	Source string

	// Subject is the finding's subject (e.g. "Pod/ns/name") → the
	// subject-scoped silence's matcher.subject. May be empty (then the
	// subject link still mints a Source-only matcher, behaving like a
	// narrower class silence); the class link never carries it.
	Subject string

	// MessagePattern, when set, is carried on BOTH links'
	// matcher.messagePattern. Optional.
	MessagePattern string

	// ShortDur is the subject-scoped silence window (e.g. 24h).
	ShortDur time.Duration

	// LongDur is the class-scoped silence window (e.g. 90d / 2160h).
	LongDur time.Duration
}

// SilenceLinks is the pair of signed one-click URLs.
type SilenceLinks struct {
	// SubjectURL snoozes THIS specific finding for ShortDur.
	SubjectURL string
	// ClassURL mutes the finding's whole class (Source) for LongDur.
	ClassURL string
}

// MintSilenceLinks produces two signed silence URLs for a finding:
//
//   - SubjectURL: scope=subject, matcher {source, subject},
//     until = now+ShortDur.
//   - ClassURL:   scope=class,   matcher {source},
//     until = now+LongDur.
//
// Each link carries a unique JTI (one link == one one-time mint). The
// link EXPIRY (token exp) is set a bit beyond the silence window so the
// link stays clickable for the lifetime of the silence it promises —
// mirroring the approve-token policy of "link valid while the action it
// proposes is still meaningful". priv/kid sign the tokens; baseURL is
// the approval-server's external base (e.g. https://cha-approve.example).
//
// Returns an error only for signing/URL failures; callers that lack a
// signer or baseURL should simply not call this (the Slack renderer
// gates on both being present and omits the links otherwise).
func MintSilenceLinks(priv ed25519.PrivateKey, kid, baseURL string, req SilenceLinkRequest, now time.Time) (SilenceLinks, error) {
	if req.Source == "" {
		return SilenceLinks{}, fmt.Errorf("ai: silence link request missing Source")
	}
	if baseURL == "" {
		return SilenceLinks{}, fmt.Errorf("ai: silence link request missing baseURL")
	}

	// The link must outlive its silence window: an operator should be
	// able to click "Silence 90d" the day before it would have expired
	// and still have the link work. We give the token exp the silence
	// window + a generous clickability buffer.
	const clickBuffer = 7 * 24 * time.Hour

	subjectClaims := SilenceTokenClaims{
		Issuer:         "cha/approval-server",
		Audience:       "cha/silence",
		JTI:            fmt.Sprintf("sil-subj-%s-%s-%d", sanitizeJTI(req.Source), sanitizeJTI(req.Subject), now.UnixNano()),
		IssuedAt:       now.Unix(),
		ExpiresAt:      now.Add(req.ShortDur + clickBuffer).Unix(),
		Scope:          SilenceScopeSubject,
		Source:         req.Source,
		Subject:        req.Subject,
		MessagePattern: req.MessagePattern,
		UntilUnix:      now.Add(req.ShortDur).Unix(),
	}
	classClaims := SilenceTokenClaims{
		Issuer:         "cha/approval-server",
		Audience:       "cha/silence",
		JTI:            fmt.Sprintf("sil-class-%s-%d", sanitizeJTI(req.Source), now.UnixNano()),
		IssuedAt:       now.Unix(),
		ExpiresAt:      now.Add(req.LongDur + clickBuffer).Unix(),
		Scope:          SilenceScopeClass,
		Source:         req.Source,
		Subject:        "", // class scope is Source-only — never carries a subject.
		MessagePattern: req.MessagePattern,
		UntilUnix:      now.Add(req.LongDur).Unix(),
	}

	subjectTok, err := SignSilenceToken(priv, kid, subjectClaims)
	if err != nil {
		return SilenceLinks{}, fmt.Errorf("ai: sign subject silence token: %w", err)
	}
	classTok, err := SignSilenceToken(priv, kid, classClaims)
	if err != nil {
		return SilenceLinks{}, fmt.Errorf("ai: sign class silence token: %w", err)
	}

	subjectURL, err := silenceURL(baseURL, subjectTok)
	if err != nil {
		return SilenceLinks{}, err
	}
	classURL, err := silenceURL(baseURL, classTok)
	if err != nil {
		return SilenceLinks{}, err
	}
	return SilenceLinks{SubjectURL: subjectURL, ClassURL: classURL}, nil
}

// silenceURL builds <baseURL>/silence?token=<jws>.
func silenceURL(baseURL, token string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("ai: invalid silence base URL: %w", err)
	}
	u.Path = joinURLPath(u.Path, "/silence")
	q := u.Query()
	q.Set("token", token)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// joinURLPath joins two URL path segments with exactly one slash.
func joinURLPath(a, b string) string {
	if a == "" {
		return b
	}
	if len(a) > 0 && a[len(a)-1] == '/' {
		a = a[:len(a)-1]
	}
	if len(b) > 0 && b[0] == '/' {
		return a + b
	}
	return a + "/" + b
}

// sanitizeJTI keeps a JTI readable + collision-resistant: non-alnum
// runs collapse to '-'. The UnixNano suffix in the caller guarantees
// uniqueness; this just makes the prefix human-greppable.
func sanitizeJTI(s string) string {
	out := make([]byte, 0, len(s))
	prevDash := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9'):
			out = append(out, c)
			prevDash = false
		case c >= 'A' && c <= 'Z':
			out = append(out, c+32)
			prevDash = false
		default:
			if !prevDash {
				out = append(out, '-')
				prevDash = true
			}
		}
	}
	return string(out)
}
