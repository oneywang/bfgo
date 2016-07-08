package qite

type Scenario struct {
	strategy *Strategy
}

func NewScenario() *Scenario {
	return &Scenario{}
}

func (s *Scenario) SetStrategy(st *Strategy) {
	//
	s.strategy = st
}

func (s *Scenario) StartStrategy() {
	//
	s.strategy.Start()
}
