package qite

type Instrument struct {
	Symbol   string
	Exchange string
}

// 用户自己直接构造即可
func NewInstrument(symbol, exchange string) *Instrument {
	return &Instrument{Symbol: symbol, Exchange: exchange}
}

// 两个Instrument指针怎么办呢？
func (i Instrument) Equal(u Instrument) bool {
	return i.Symbol == u.Symbol && i.Exchange == u.Exchange
}

type InstrumentManager struct {
	intruments []Instrument
}
