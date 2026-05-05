package client

import (
	"testing"

	"github.com/svinson1121/vectorcore-smsc/internal/smpp"
	"github.com/svinson1121/vectorcore-smsc/internal/store"
)

func TestApplyAddressOverridesUsesConfiguredTONNPI(t *testing.T) {
	sourceTON, sourceNPI := 5, 0
	destTON, destNPI := 2, 8
	mgr := &Manager{
		sessions: map[string]*Session{
			"client-1": {
				cfg: store.SMPPClient{
					Name:          "peer-a",
					SystemID:      "system-a",
					SourceAddrTON: &sourceTON,
					SourceAddrNPI: &sourceNPI,
					DestAddrTON:   &destTON,
					DestAddrNPI:   &destNPI,
				},
			},
		},
	}
	pdu := &smpp.PDU{
		CommandID:     smpp.CmdDeliverSM,
		SourceAddrTON: 1,
		SourceAddrNPI: 1,
		DestAddrTON:   1,
		DestAddrNPI:   1,
	}

	mgr.applyAddressOverrides("peer-a", pdu)

	if pdu.SourceAddrTON != 5 || pdu.SourceAddrNPI != 0 || pdu.DestAddrTON != 2 || pdu.DestAddrNPI != 8 {
		t.Fatalf("unexpected TON/NPI overrides: src=%d/%d dst=%d/%d",
			pdu.SourceAddrTON, pdu.SourceAddrNPI, pdu.DestAddrTON, pdu.DestAddrNPI)
	}
}

func TestApplyAddressOverridesAutoPreservesPDUValues(t *testing.T) {
	mgr := &Manager{
		sessions: map[string]*Session{
			"client-1": {
				cfg: store.SMPPClient{
					Name:     "peer-a",
					SystemID: "system-a",
				},
			},
		},
	}
	pdu := &smpp.PDU{
		CommandID:     smpp.CmdDeliverSM,
		SourceAddrTON: 1,
		SourceAddrNPI: 1,
		DestAddrTON:   0,
		DestAddrNPI:   1,
	}

	mgr.applyAddressOverrides("peer-a", pdu)

	if pdu.SourceAddrTON != 1 || pdu.SourceAddrNPI != 1 || pdu.DestAddrTON != 0 || pdu.DestAddrNPI != 1 {
		t.Fatalf("auto should preserve PDU TON/NPI, got src=%d/%d dst=%d/%d",
			pdu.SourceAddrTON, pdu.SourceAddrNPI, pdu.DestAddrTON, pdu.DestAddrNPI)
	}
}
