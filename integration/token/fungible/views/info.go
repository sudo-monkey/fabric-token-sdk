/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

type GetEnrollmentID struct {
	Wallet string
	TMSID  token.TMSID
}

// GetEnrollmentIDView is a view that returns the enrollment ID of a wallet.
type GetEnrollmentIDView struct {
	*GetEnrollmentID
}

func (r *GetEnrollmentIDView) Call(context view.Context) (interface{}, error) {
	tms := token.GetManagementService(context, token.WithTMSID(r.TMSID))
	assert.NotNil(tms, "tms not found [%s]", r.TMSID)
	w := tms.WalletManager().OwnerWallet(r.Wallet)
	assert.NotNil(w, "wallet not found [%s]", r.Wallet)
	return w.EnrollmentID(), nil
}

type GetEnrollmentIDViewFactory struct{}

func (p *GetEnrollmentIDViewFactory) NewView(in []byte) (view.View, error) {
	f := &GetEnrollmentIDView{GetEnrollmentID: &GetEnrollmentID{}}
	err := json.Unmarshal(in, f.GetEnrollmentID)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}

type CheckPublicParamsMatch struct {
	TMSID token.TMSID
}

type CheckPublicParamsMatchView struct {
	*CheckPublicParamsMatch
}

func (p *CheckPublicParamsMatchView) Call(context view.Context) (interface{}, error) {
	tms := token.GetManagementService(context, token.WithTMSID(p.TMSID))
	assert.NotNil(tms, "failed to get TMS")

	assert.NoError(tms.PublicParametersManager().Validate(), "failed to validate local public parameters")

	ppRaw, err := tms.PublicParametersManager().SerializePublicParameters()
	assert.NoError(err, "failed to marshal public params")

	fetchedPPRaw, err := tms.PublicParametersManager().Fetch()
	assert.NoError(err, "failed to fetch public params")

	ppm, err := token.NewPublicParametersManagerFromPublicParams(fetchedPPRaw)
	assert.NoError(err, "failed to instantiate public params manager from fetch params")
	assert.NoError(ppm.Validate(), "failed to validate remote public parameters")

	assert.Equal(fetchedPPRaw, ppRaw, "public params do not match [%s]!=[%s]", hash.Hashable(fetchedPPRaw), hash.Hashable(ppRaw))

	return nil, nil
}

type CheckPublicParamsMatchViewFactory struct{}

func (p *CheckPublicParamsMatchViewFactory) NewView(in []byte) (view.View, error) {
	f := &CheckPublicParamsMatchView{CheckPublicParamsMatch: &CheckPublicParamsMatch{}}
	err := json.Unmarshal(in, f.CheckPublicParamsMatch)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}
