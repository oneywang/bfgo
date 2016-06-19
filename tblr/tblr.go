package main

import . "github.com/sunwangme/bfgo/api/bfgateway"
import . "github.com/sunwangme/bfgo/api/bfdatafeed"

// 本策略的交易参数常量
const (
	TRADE_VOLUME int32  = 1
	VOLUME_LIMIT int32  = 5
	FAST_K_NUM   uint32 = 15
	SLOW_K_NUM   uint32 = 60
)

// 本策略的变量
var _period BfBarPeriod = BfBarPeriod_PERIOD_M01
var _historyBarsGot bool = false
var _barsCount uint32 = 0
var _currentBarMinute uint32 = 0
var _fastMa, _slowMa []float64
var _fastMa0, _fastMa1, _slowMa0, _slowMa1 float64 = 0, 0, 0, 0
var _positionLong, _positionShort int32 = 0, 0
var _pendingOrderIds []string

func initPosition(position *BfPositionData) {
	if _positionLong > 0 || _positionShort > 0 {
		// already inited
		return
	}
	if position.Direction == BfDirection_DIRECTION_LONG {
		_positionLong += position.Position
	} else if position.Direction == BfDirection_DIRECTION_SHORT {
		_positionShort += position.Position
	}
}

func updatePosition(direction BfDirection, offset BfOffset, volume int32) {
	if direction == BfDirection_DIRECTION_LONG && offset == BfOffset_OFFSET_OPEN {
		_positionLong += volume
	} else if direction == BfDirection_DIRECTION_LONG && offset == BfOffset_OFFSET_CLOSE {
		_positionLong -= volume
	} else if direction == BfDirection_DIRECTION_SHORT && offset == BfOffset_OFFSET_OPEN {
		_positionShort += volume
	} else if direction == BfDirection_DIRECTION_SHORT && offset == BfOffset_OFFSET_CLOSE {
		_positionShort -= volume
	}
}

func indexOf(a []string, v string) int {
	for i := range a {
		if a[i] == v {
			return i
		}
	}
	return -1
}

func without(a []string, v string) []string {
	var r []string
	j := 0
	for i := range a {
		if a[i] != v {
			r[j] = a[i]
			j++
		}
	}
	return r
}
