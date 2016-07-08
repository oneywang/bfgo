package qite

import . "github.com/sunwangme/bfgo/bftraderclient"

type Framework struct {
	client *BfTrderClient
	*InstrumentManager
}

func newFramework() *Framework {
	return &Framework{}
}

var CurrentFramework = newFramework()

func (f *Framework) SetClient(client *BfTrderClient) {
	f.client = client
}
