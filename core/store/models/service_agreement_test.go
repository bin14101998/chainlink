package models_test

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/smartcontractkit/chainlink/core/internal/cltest"
	"github.com/smartcontractkit/chainlink/core/store/assets"
	"github.com/smartcontractkit/chainlink/core/store/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUnsignedServiceAgreementFromRequest(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		input          string
		wantDigest     string
		wantPayment    *assets.Link
		wantAggregator string
	}{
		{
			"basic",
			`{"payment":"1","initiators":[{"type":"web"}],"tasks":[` +
				`{"type":"httpget","url":"https://bitstamp.net/api/ticker/"},` +
				`{"type":"jsonparse","path":["last"]},` +
				`{"type":"ethbytes32"},{"type":"ethtx"}],` +
				`"aggregator":"0xDeaDbeefdEAdbeefdEadbEEFdeadbeEFdEaDbeeF"}`,
			"0x9ef887cb4777a861deccf0ad5c9b862bff25226986732b4f400b4b8cfa2bf027",
			assets.NewLink(1),
			"0xDeaDbeefdEAdbeefdEadbEEFdeadbeEFdEaDbeeF",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var sar models.JobSpecRequest
			require.NoError(t, json.Unmarshal([]byte(test.input), &sar))
			us, err := models.NewUnsignedServiceAgreementFromRequest(strings.NewReader(test.input))
			require.NoError(t, err)
			assert.Equal(t, test.wantDigest, us.ID.String())
			assert.Equal(t, test.wantPayment, us.Encumbrance.Payment)
			assert.Equal(t, test.wantAggregator, us.Encumbrance.Aggregator.String())
			assert.Equal(t, cltest.NormalizedJSON(t, []byte(test.input)), us.RequestBody)
		})
	}
}

func TestBuildServiceAgreement(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		input          string
		wantDigest     string
		wantPayment    *assets.Link
		wantAggregator string
	}{
		{
			"basic",
			`{"payment":"1","initiators":[{"type":"web"}],"tasks":[` +
				`{"type":"httpget","url":"https://bitstamp.net/api/ticker/"},` +
				`{"type":"jsonparse","path":["last"]},` +
				`{"type":"ethbytes32"},{"type":"ethtx"}],` +
				`"aggregator":"0xDeaDbeefdEAdbeefdEadbEEFdeadbeEFdEaDbeeF"}`,
			"0x9ef887cb4777a861deccf0ad5c9b862bff25226986732b4f400b4b8cfa2bf027",
			assets.NewLink(1),
			"0xDeaDbeefdEAdbeefdEadbEEFdeadbeEFdEaDbeeF",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var sar models.JobSpecRequest
			assert.NoError(t, json.Unmarshal([]byte(test.input), &sar))

			us, err := models.NewUnsignedServiceAgreementFromRequest(strings.NewReader(test.input))
			assert.NoError(t, err)

			sa, err := models.BuildServiceAgreement(us, cltest.MockSigner{})
			require.NoError(t, err)
			assert.Equal(t, test.wantDigest, sa.ID)
			assert.Equal(t, test.wantPayment, sa.Encumbrance.Payment)
			assert.Equal(t, test.wantAggregator, sa.Encumbrance.Aggregator.String())
			assert.Equal(t, cltest.NormalizedJSON(t, []byte(test.input)), sa.RequestBody)
			assert.NotEqual(t, models.AnyTime{}, sa.CreatedAt)
			assert.NotEqual(t, "", sa.Signature.String())
		})
	}
}

func TestEncumbrance_ABI(t *testing.T) {
	t.Parallel()
	endAt, _ := time.Parse("2006-01-02T15:04:05.000Z", "2007-01-02T15:04:05.000Z")
	deadbeef, err := strconv.ParseInt("deadbeef", 16, 64)
	require.NoError(t, err)
	tests := []struct {
		name       string
		payment    *assets.Link
		expiration int
		endAt      models.AnyTime
		aggregator models.EIP55Address
		oracles    []models.EIP55Address
		want       string
	}{
		{"basic", assets.NewLink(1), 2, models.AnyTime{},
			models.EIP55Address("0x0000000000000000000000000000000000000000000000000000000000000000"),
			[]models.EIP55Address{},
			"0x0000000000000000000000000000000000000000000000000000000000000001" +
				"0000000000000000000000000000000000000000000000000000000000000002" +
				"886e0900" +
				"0000000000000000000000000000000000000000000000000000000000000000"},
		{"basic dead beef payment", assets.NewLink(deadbeef), 2, models.AnyTime{},
			models.EIP55Address("0x0000000000000000000000000000000000000000000000000000000000000000"),
			[]models.EIP55Address{},
			"0x00000000000000000000000000000000000000000000000000000000deadbeef" +
				"0000000000000000000000000000000000000000000000000000000000000002" +
				"886e0900" +
				"0000000000000000000000000000000000000000000000000000000000000000"},
		{"oracle address", nil, 0, models.AnyTime{},
			models.EIP55Address("0xDeaDbeefdEAdbeefdEadbEEFdeadbeEFdEaDbeeF"),
			[]models.EIP55Address{
				models.EIP55Address("0xa0788FC17B1dEe36f057c42B6F373A34B014687e")},
			"0x0000000000000000000000000000000000000000000000000000000000000000" +
				"0000000000000000000000000000000000000000000000000000000000000000" +
				"886e0900" +
				"000000000000000000000000deadbeefdeadbeefdeadbeefdeadbeefdeadbeef" +
				"000000000000000000000000a0788fc17b1dee36f057c42b6f373a34b014687e"},
		{"different endAt", nil, 0, models.NewAnyTime(endAt),
			models.EIP55Address("0xDeaDbeefdEAdbeefdEadbEEFdeadbeEFdEaDbeeF"),
			[]models.EIP55Address{models.EIP55Address("0xa0788FC17B1dEe36f057c42B6F373A34B014687e")},
			"0x0000000000000000000000000000000000000000000000000000000000000000" +
				"0000000000000000000000000000000000000000000000000000000000000000" +
				"459a7465" +
				"000000000000000000000000deadbeefdeadbeefdeadbeefdeadbeefdeadbeef" +
				"000000000000000000000000a0788fc17b1dee36f057c42b6f373a34b014687e"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			enc := models.Encumbrance{
				Payment:    test.payment,
				Expiration: uint64(test.expiration),
				EndAt:      test.endAt,
				Aggregator: test.aggregator,
				Oracles:    test.oracles,
			}

			ebytes, err := enc.ABI()
			assert.NoError(t, err)
			assert.Equal(t, test.want, hexutil.Encode(ebytes))
		})
	}
}
