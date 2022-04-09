/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

func TestRequestTransferIssue(t *testing.T) {
	r := NewRequest(nil, "hello world")

	_, err := r.Issue(nil, []byte("alice"), "tok", 1)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "wallet is nil")

	_, err = r.Issue(&IssuerWallet{}, []byte{}, "tok", 1)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "receiver is nil")

	_, err = r.Issue(&IssuerWallet{}, []byte("alice"), "", 1)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "type is empty")

	_, err = r.Issue(&IssuerWallet{}, []byte("alice"), "tok", 0)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "quantity is 0")
}

func TestRequestTransferInput(t *testing.T) {
	r := NewRequest(nil, "hello world")

	_, err := r.Transfer(nil, "hello world", []uint64{1}, []view.Identity{[]byte("alice")})
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "wallet is nil")

	_, err = r.Transfer(&OwnerWallet{}, "", []uint64{1}, []view.Identity{[]byte("alice")})
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "type is empty")

	_, err = r.Transfer(&OwnerWallet{}, "hello world", nil, []view.Identity{[]byte("alice")})
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "values is empty")

	_, err = r.Transfer(&OwnerWallet{}, "hello world", []uint64{1}, []view.Identity{})
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "owners is empty")

	_, err = r.Transfer(&OwnerWallet{}, "hello world", []uint64{1, 2}, []view.Identity{[]byte("alice")})
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "values and owners must have the same length")

	_, err = r.Transfer(&OwnerWallet{}, "hello world", []uint64{1, 0}, []view.Identity{[]byte("alice"), []byte("bob")})
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "values must be non-zero")

	_, err = r.Transfer(&OwnerWallet{}, "hello world", []uint64{1, 2}, []view.Identity{[]byte("alice"), []byte{}})
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "all recipients should be defined")
}

func TestRequestTransferRedeem(t *testing.T) {
	r := NewRequest(nil, "hello world")

	err := r.Redeem(nil, "tok", 1)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "wallet is nil")

	err = r.Redeem(&OwnerWallet{}, "", 1)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "type is empty")

	err = r.Redeem(&OwnerWallet{}, "tok", 0)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "quantity is 0")
}

func TestRequestSerialization(t *testing.T) {
	r := NewRequest(nil, "hello world")
	r.Actions = &driver.TokenRequest{
		Issues: [][]byte{
			[]byte("issue1"),
			[]byte("issue2"),
		},
		Transfers:         [][]byte{[]byte("transfer1")},
		Signatures:        [][]byte{[]byte("signature1")},
		AuditorSignatures: [][]byte{[]byte("auditor_signature1")},
	}
	r.Metadata = &driver.TokenRequestMetadata{
		Issues:      []driver.IssueMetadata{},
		Transfers:   nil,
		Application: nil,
	}
	raw, err := r.Bytes()
	assert.NoError(t, err)

	r2 := NewRequest(nil, "")
	err = r2.FromBytes(raw)
	assert.NoError(t, err)
	raw2, err := r2.Bytes()
	assert.NoError(t, err)

	assert.Equal(t, raw, raw2)

	mRaw, err := r.MarshallToAudit()
	assert.NoError(t, err)
	mRaw2, err := r2.MarshallToAudit()
	assert.NoError(t, err)
	assert.Equal(t, mRaw, mRaw2)

	mRaw, err = r.MarshallToSign()
	assert.NoError(t, err)
	mRaw2, err = r2.MarshallToSign()
	assert.NoError(t, err)
	assert.Equal(t, mRaw, mRaw2)

	mRaw, err = r.ActionsToBytes()
	assert.NoError(t, err)
	mRaw2, err = r2.ActionsToBytes()
	assert.NoError(t, err)
	assert.Equal(t, mRaw, mRaw2)

}

func TestImport(t *testing.T) {
	r1 := NewRequest(nil, "hello world")
	r2 := NewRequest(nil, "hello nations")

	err := r2.Import(r1)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "cannot import request with different anchor")
}
