package api

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/spiffe/spire-api-sdk/proto/spire/api/types"
	"github.com/spiffe/spire/pkg/common/idutil"
	"github.com/spiffe/spire/proto/spire/common"
)

func BundleToProto(b *common.Bundle) (*types.Bundle, error) {
	if b == nil {
		return nil, errors.New("no bundle provided")
	}

	td, err := idutil.TrustDomainFromString(b.TrustDomainId)
	if err != nil {
		return nil, err
	}

	return &types.Bundle{
		TrustDomain:     td.String(),
		RefreshHint:     b.RefreshHint,
		SequenceNumber:  0,
		X509Authorities: CertificatesToProto(b.RootCas),
		JwtAuthorities:  PublicKeysToProto(b.JwtSigningKeys),
	}, nil
}

func CertificatesToProto(rootCas []*common.Certificate) []*types.X509Certificate {
	var x509Authorities []*types.X509Certificate
	for _, rootCA := range rootCas {
		x509Authorities = append(x509Authorities, &types.X509Certificate{
			Asn1: rootCA.DerBytes,
		})
	}

	return x509Authorities
}
func PublicKeysToProto(keys []*common.PublicKey) []*types.JWTKey {
	var jwtAuthorities []*types.JWTKey
	for _, key := range keys {
		jwtAuthorities = append(jwtAuthorities, &types.JWTKey{
			PublicKey: key.PkixBytes,
			KeyId:     key.Kid,
			ExpiresAt: key.NotAfter,
		})
	}
	return jwtAuthorities
}

func ProtoToBundle(b *types.Bundle) (*common.Bundle, error) {
	if b == nil {
		return nil, errors.New("no bundle provided")
	}

	td, err := idutil.TrustDomainFromString(b.TrustDomain)
	if err != nil {
		return nil, err
	}

	rootCas, err := ParseX509Authorities(b.X509Authorities)
	if err != nil {
		return nil, fmt.Errorf("unable to parse X.509 authority: %w", err)
	}

	jwtSigningKeys, err := ParseJWTAuthorities(b.JwtAuthorities)
	if err != nil {
		return nil, fmt.Errorf("unable to parse JWT authority: %w", err)
	}

	commonBundle := &common.Bundle{
		TrustDomainId:  td.IDString(),
		RefreshHint:    b.RefreshHint,
		RootCas:        rootCas,
		JwtSigningKeys: jwtSigningKeys,
	}

	return commonBundle, nil
}

func ProtoToBundleMask(mask *types.BundleMask) *common.BundleMask {
	if mask == nil {
		return nil
	}

	return &common.BundleMask{
		JwtSigningKeys: mask.JwtAuthorities,
		RootCas:        mask.X509Authorities,
		RefreshHint:    mask.RefreshHint,
	}
}

func ParseX509Authorities(certs []*types.X509Certificate) ([]*common.Certificate, error) {
	var rootCAs []*common.Certificate
	for _, rootCA := range certs {
		if _, err := x509.ParseCertificates(rootCA.Asn1); err != nil {
			return nil, err
		}

		rootCAs = append(rootCAs, &common.Certificate{
			DerBytes: rootCA.Asn1,
		})
	}

	return rootCAs, nil
}

func ParseJWTAuthorities(keys []*types.JWTKey) ([]*common.PublicKey, error) {
	var jwtKeys []*common.PublicKey
	for _, key := range keys {
		if _, err := x509.ParsePKIXPublicKey(key.PublicKey); err != nil {
			return nil, err
		}

		if key.KeyId == "" {
			return nil, errors.New("missing key ID")
		}

		jwtKeys = append(jwtKeys, &common.PublicKey{
			PkixBytes: key.PublicKey,
			Kid:       key.KeyId,
			NotAfter:  key.ExpiresAt,
		})
	}

	return jwtKeys, nil
}

func HashByte(b []byte) string {
	if len(b) == 0 {
		return ""
	}

	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}
