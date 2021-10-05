/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package sigproof

import (
	"encoding/json"

	"github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/pssign"
	"github.com/pkg/errors"
)

// membership proof based on ps signature
type MembershipProof struct {
	Challenge         *math.Zr
	Signature         *pssign.Signature
	Value             *math.Zr
	ComBlindingFactor *math.Zr
	SigBlindingFactor *math.Zr
	Hash              *math.Zr
	Commitment        *math.G1
}

func (p *MembershipProof) Serialize() ([]byte, error) {
	return json.Marshal(p)
}

func (p *MembershipProof) Deserialize(raw []byte) error {
	return json.Unmarshal(raw, p)
}

// witness for membership proof
type MembershipWitness struct {
	signature         *pssign.Signature
	value             *math.Zr
	hash              *math.Zr
	sigBlindingFactor *math.Zr
	comBlidingFactor  *math.Zr
}

func NewMembershipWitness(sig *pssign.Signature, value *math.Zr, bf *math.Zr) *MembershipWitness {
	return &MembershipWitness{signature: sig, value: value, comBlidingFactor: bf}
}

func NewMembershipProver(witness *MembershipWitness, com, P *math.G1, Q *math.G2, PK []*math.G2, pp []*math.G1, curve *math.Curve) *MembershipProver {
	return &MembershipProver{witness: witness, MembershipVerifier: NewMembershipVerifier(com, P, Q, PK, pp, curve)}
}

func NewMembershipVerifier(com, P *math.G1, Q *math.G2, PK []*math.G2, pp []*math.G1, curve *math.Curve) *MembershipVerifier {
	return &MembershipVerifier{PedersenParams: pp, CommitmentToValue: com, POKVerifier: &POKVerifier{PK: PK, Q: Q, P: P, Curve: curve}}
}

// prover
type MembershipProver struct {
	*MembershipVerifier
	witness    *MembershipWitness
	randomness *MembershipRandomness
	Commitment *MembershipCommitment
}

// MembershipCommitment to randomness in proof
type MembershipCommitment struct {
	CommitmentToValue *math.G1
	Signature         *math.Gt
}

// MembershipRandomness used in proof
type MembershipRandomness struct {
	value             *math.Zr
	comBlindingFactor *math.Zr
	sigBlindingFactor *math.Zr
	hash              *math.Zr
}

// verify whether a value has been signed
type MembershipVerifier struct {
	*POKVerifier
	PedersenParams    []*math.G1
	CommitmentToValue *math.G1
}

// generate a membership proof
func (p *MembershipProver) Prove() ([]byte, error) {
	if len(p.PK) != 3 {
		return nil, errors.Errorf("can't generate membership proof")
	}
	if len(p.PedersenParams) != 2 {
		return nil, errors.Errorf("can't generate membership proof")
	}
	proof := &MembershipProof{}
	proof.Commitment = p.CommitmentToValue
	// obfuscate signature
	var err error
	proof.Signature, err = p.obfuscateSignature()
	if err != nil {
		return nil, err
	}
	// compute hash
	p.computeHash()
	// compute commitment
	err = p.computeCommitment()
	if err != nil {
		return nil, err
	}
	// compute challenge
	proof.Challenge, err = p.computeChallenge(proof.Commitment, p.Commitment, proof.Signature)
	if err != nil {
		return nil, err
	}

	// generate proof
	sp := &common.SchnorrProver{Witness: []*math.Zr{p.witness.value, p.witness.comBlidingFactor, p.witness.hash, p.witness.sigBlindingFactor}, Randomness: []*math.Zr{p.randomness.value, p.randomness.comBlindingFactor, p.randomness.hash, p.randomness.sigBlindingFactor}, Challenge: proof.Challenge, SchnorrVerifier: &common.SchnorrVerifier{Curve: p.Curve}}
	proofs, err := sp.Prove()
	if err != nil {
		return nil, errors.Wrapf(err, "range proof generation failed")
	}
	proof.Value = proofs[0]
	proof.ComBlindingFactor = proofs[1]
	proof.Hash = proofs[2]
	proof.SigBlindingFactor = proofs[3]

	return proof.Serialize()
}

// verify membership proof
func (v *MembershipVerifier) Verify(raw []byte) error {
	if len(v.PK) != 3 {
		return errors.Errorf("can't generate membership proof")
	}
	if len(v.PedersenParams) != 2 {
		return errors.Errorf("can't generate membership proof")
	}
	proof := &MembershipProof{}
	err := proof.Deserialize(raw)
	if err != nil {
		return err
	}

	com, err := v.recomputeCommitments(proof)
	if err != nil {
		return err
	}

	chal, err := v.computeChallenge(proof.Commitment, com, proof.Signature)
	if err != nil {
		return err
	}
	if !chal.Equals(proof.Challenge) {
		return errors.Errorf("invalid membership proof")
	}
	return nil
}

func (p *MembershipProver) obfuscateSignature() (*pssign.Signature, error) {
	rand, err := p.Curve.Rand()
	if err != nil {
		return nil, errors.Errorf("failed to get RNG")
	}

	p.witness.sigBlindingFactor = p.Curve.NewRandomZr(rand)
	v := pssign.NewVerifier(nil, nil, p.Curve)
	err = v.Randomize(p.witness.signature)
	if err != nil {
		return nil, err
	}
	sig := &pssign.Signature{}
	sig.Copy(p.witness.signature)
	sig.S.Add(p.P.Mul(p.witness.sigBlindingFactor))

	return sig, nil
}

func (p *MembershipProver) computeCommitment() error {
	// Get RNG
	rand, err := p.Curve.Rand()
	if err != nil {
		return errors.Errorf("failed to get RNG")
	}
	// compute commitments
	p.randomness = &MembershipRandomness{}
	p.randomness.value = p.Curve.NewRandomZr(rand)
	p.randomness.hash = p.Curve.NewRandomZr(rand)
	p.randomness.sigBlindingFactor = p.Curve.NewRandomZr(rand)

	t := p.PK[1].Mul(p.randomness.value)
	t.Add(p.PK[2].Mul(p.randomness.hash))

	p.Commitment = &MembershipCommitment{}
	p.Commitment.Signature = p.Curve.Pairing2(t, p.witness.signature.R, p.Q, p.P.Mul(p.randomness.sigBlindingFactor))
	p.Commitment.Signature = p.Curve.FExp(p.Commitment.Signature)

	p.randomness.comBlindingFactor = p.Curve.NewRandomZr(rand)
	p.Commitment.CommitmentToValue = p.PedersenParams[0].Mul(p.randomness.value)
	p.Commitment.CommitmentToValue.Add(p.PedersenParams[1].Mul(p.randomness.comBlindingFactor))

	return nil
}

func (v *MembershipVerifier) computeChallenge(comToValue *math.G1, com *MembershipCommitment, signature *pssign.Signature) (*math.Zr, error) {
	g1array := common.GetG1Array(v.PedersenParams, []*math.G1{comToValue, com.CommitmentToValue, v.P})
	g2array := common.GetG2Array(v.PK, []*math.G2{v.Q})
	raw := common.GetBytesArray(g1array.Bytes(), g2array.Bytes(), com.Signature.Bytes())
	bytes, err := signature.Serialize()
	if err != nil {
		return nil, errors.Errorf("failed to compute challenge: error while serializing Pointcheval-Sanders signature")
	}
	raw = append(raw, bytes...)

	return v.Curve.HashToZr(raw), nil
}

func (p *MembershipProver) computeHash() {
	bytes := p.witness.value.Bytes()
	p.witness.hash = p.Curve.HashToZr(bytes)
	return
}

// recompute commitments for verification
func (v *MembershipVerifier) recomputeCommitments(p *MembershipProof) (*MembershipCommitment, error) {
	psv := &POKVerifier{P: v.P, Q: v.Q, PK: v.PK, Curve: v.Curve}
	c := &MembershipCommitment{}

	psp := &POK{
		Challenge:      p.Challenge,
		Signature:      p.Signature,
		Messages:       []*math.Zr{p.Value},
		Hash:           p.Hash,
		BlindingFactor: p.SigBlindingFactor,
	}
	var err error
	c.Signature, err = psv.RecomputeCommitment(psp)
	if err != nil {
		return nil, err
	}
	ver := &common.SchnorrVerifier{PedParams: v.PedersenParams, Curve: v.Curve}
	zkp := &common.SchnorrProof{Statement: v.CommitmentToValue, Proof: []*math.Zr{p.Value, p.ComBlindingFactor}, Challenge: p.Challenge}
	c.CommitmentToValue = ver.RecomputeCommitment(zkp)

	return c, nil
}
