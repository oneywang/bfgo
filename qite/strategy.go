package qite

type Strategy struct {
	Bars        *BarSeries
	Instruments []Instrument
}

func NewStrategy() *Strategy {
	return &Strategy{}
}

func (s *Strategy) AddInstrument(i Instrument) {
	s.Instruments = append(s.Instruments, i)
}

func (s *Strategy) Start() {
	//TODO
}
