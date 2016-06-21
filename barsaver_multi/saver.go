package main

import . "github.com/sunwangme/bfgo/api/bfgateway"
import . "github.com/sunwangme/bfgo/api/bfdatafeed"
import "github.com/sunwangme/bfgo/oneywang/bar"

// 保存bar所用的核心数据结构
type BarSlice map[BfBarPeriod]*BfBarData
type Saver struct {
	// 不同品种当前的1分钟K线
	data map[string]*BarSlice
}

func NewSaver() *Saver {
	this := &Saver{data: make(map[string]*BarSlice)}
	return this
}

// 将tick保存到Converter内置的某周期的bar
// 返回值
// bool：是否新周期开始
// *BfBarData：如果新周期开始，返回的是上一周期的bar；否则是更新后的老bar
func (p *Saver) SaveTick2Bar(tick *BfTickData, period BfBarPeriod) (*BfBarData, bool) {
	var ret *BfBarData = nil
	isNewPeriod := false

	id := tick.Symbol + "@" + tick.Exchange
	d, ok := p.data[id]
	if !ok {
		// 这个品种第一次赋值&1分钟的第一次赋值，生成barSlice
		var bs BarSlice = make(map[BfBarPeriod]*BfBarData)
		p.data[id] = &bs
		d = &bs
	}

	if storedBar, ok := (*d)[period]; !ok {
		// 这个周期的bar第一次赋值
		ret = bar.NewBarFromTick(tick, period)
		(*d)[period] = ret
	} else {
		ret, isNewPeriod = bar.UpdateTick2Bar(tick, storedBar)
		if isNewPeriod {
			// 新的周期开始，需要返回老bar以便插入db，同时保存新周期的bar
			(*d)[period] = ret
		}
		ret = storedBar
	}

	return ret, isNewPeriod
}
