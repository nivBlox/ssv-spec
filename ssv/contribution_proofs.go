package ssv

import (
	"encoding/hex"
	spec "github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/bloxapp/ssv-spec/types"
	"github.com/pkg/errors"
)

func (dr *Runner) SignSyncSubCommitteeContributionProof(slot spec.Slot, indexes []uint64, signer types.KeyManager) (*PartialSignatureMessages, error) {
	ret := make([]*PartialSignatureMessage, 0)
	for _, index := range indexes {
		sig, r, err := signer.SignContributionProof(slot, index, dr.Share.SharePubKey)
		if err != nil {
			return nil, errors.Wrap(err, "could not sign partial selection proof")
		}
		r = ensureRoot(r)
		// TODO what guarantees the order here? Is it important?
		ret = append(ret, &PartialSignatureMessage{
			Slot:             slot,
			PartialSignature: sig,
			SigningRoot:      r,
			Signer:           dr.Share.OperatorID,
			MetaData: &PartialSignatureMetaData{
				ContributionSubCommitteeIndex: index,
			},
		})
	}
	return &PartialSignatureMessages{
		Type:     ContributionProofs,
		Messages: ret,
	}, nil
}

// ProcessContributionProofsMessage process contribution proofs msg (an array), returns true if it has quorum for partial signatures.
// returns true only once (first time quorum achieved)
func (dr *Runner) ProcessContributionProofsMessage(signedMsg *SignedPartialSignatureMessage) (bool, [][]byte, error) {
	if err := dr.canProcessContributionProofMsg(signedMsg); err != nil {
		return false, nil, errors.Wrap(err, "selection proof msg invalid")
	}

	roots := make([][]byte, 0)
	anyQuorum := false
	for _, msg := range signedMsg.Message.Messages {
		prevQuorum := dr.State.ContributionProofs.HasQuorum(msg.SigningRoot)

		if err := dr.State.ContributionProofs.AddSignature(msg); err != nil {
			return false, nil, errors.Wrap(err, "could not add partial contribution proof signature")
		}
		// TODO need to check roots are unique and not overriding.. we are getting multiple messages here
		dr.State.ContributionSubCommitteeIndexes[hex.EncodeToString(msg.SigningRoot)] = msg.MetaData.ContributionSubCommitteeIndex

		if prevQuorum {
			continue
		}

		quorum := dr.State.ContributionProofs.HasQuorum(msg.SigningRoot)
		if quorum {
			roots = append(roots, msg.SigningRoot)
			anyQuorum = true
		}
	}

	return anyQuorum, roots, nil
}

// validateRandaoMsg returns true if it can process selection proof message, false if not
func (dr *Runner) canProcessContributionProofMsg(msg *SignedPartialSignatureMessage) error {
	return dr.validatePartialSigMsg(msg, dr.CurrentDuty.Slot)
}
