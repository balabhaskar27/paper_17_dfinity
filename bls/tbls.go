package bls

import (
	"errors"

	"gopkg.in/dedis/crypto.v0/abstract"
	"gopkg.in/dedis/crypto.v0/share"
)

type DistKeyShare interface {
	PriShare() *share.PriShare
	Polynomial() *share.PubPoly
}

type ThresholdSig struct {
	Index int
	Sig   abstract.Point
}

// ThresholdSign generates the regular BLS signature and also computes a
// discrete log equality proof to show that the signature have been correctly
// generated from the private share generated during a DKG.
func ThresholdSign(s PairingSuite, d DistKeyShare, msg []byte) *ThresholdSig {
	// sig = H(m) * x_i in G1
	HM := hashed(s, msg)
	xHM := HM.Mul(HM, d.PriShare().V)

	return &ThresholdSig{
		Index: d.PriShare().I,
		Sig:   xHM,
	}

}

// ThresholdVerify verifies that the threshold signature is have been correctly
// generated from the private share generated during a DKG.
func ThresholdVerify(s PairingSuite, public *share.PubPoly, msg []byte, sig *ThresholdSig) bool {
	HM := hashed(s, msg)
	// e(H(m) * xi, G2)
	eXHM := s.GT().PointGT().Pairing(sig.Sig, s.G2().Point().Base())
	// e(G1, G2 * xi)
	xiG := public.Eval(sig.Index).V
	exiG := s.GT().PointGT().Pairing(HM, xiG)

	return eXHM.Equal(exiG)
}

func AggregateSignatures(s PairingSuite, public *share.PubPoly, msg []byte, sigs []*ThresholdSig, n, t int) ([]byte, error) {
	pubShares := make([]*share.PubShare, 0, n)
	for _, sig := range sigs {
		if !ThresholdVerify(s, public, msg, sig) {
			continue
		}
		pubShares = append(pubShares, &share.PubShare{V: sig.Sig, I: sig.Index})
		if len(pubShares) >= t {
			break
		}
	}

	if len(pubShares) < t {
		return nil, errors.New("not enough valid threshold bls signatures")
	}

	sig, err := share.RecoverCommit(s.G1(), pubShares, t, n)
	if err != nil {
		return nil, err
	}
	buff, _ := sig.MarshalBinary()
	if err := Verify(s, public.Commit(), msg, buff); err != nil {
		panic("math is wrong?")
	}
	return buff, nil
}
