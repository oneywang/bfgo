package bar

import (
	"log"
	"math"
	"strconv"
	"strings"
)
import . "github.com/sunwangme/bfgo/api/bfgateway"
import . "github.com/sunwangme/bfgo/api/bfdatafeed"

// 支持这些周期的bar计算
var PeriodKeyList = []BfBarPeriod{
	BfBarPeriod_PERIOD_M01,
	BfBarPeriod_PERIOD_M03,
	BfBarPeriod_PERIOD_M15,
	BfBarPeriod_PERIOD_H01,
	BfBarPeriod_PERIOD_D01}

var periodMinutesList = map[BfBarPeriod]int32{
	BfBarPeriod_PERIOD_M01: 1,
	BfBarPeriod_PERIOD_M03: 3,
	BfBarPeriod_PERIOD_M15: 15}

// "%H:%M:%S.%f"==>"%H:%M:%S"
func ticktime2Bartime(t string, period BfBarPeriod) string {
	var b string

	if dot := strings.LastIndex(t, "."); dot >= 0 {
		b = t[:dot]
		//TODO: 不同period的bar，其bartime应与Period匹配!
	} else {
		log.Fatalf("Failed ticktime2bartime : %s", t)
	}
	return b
}

// 输入："%H:%M:%S"
// 输出：M值
func _bartime2Minute(t string) int32 {
	var m int32
	if strings.Count(t, ":") != 2 {
		log.Fatalf("Failed bartime2minute : %s", t)
	}
	start := strings.Index(t, ":")
	stop := strings.LastIndex(t, ":")
	if stop > start {
		i, err := strconv.Atoi(t[start+1 : stop])
		if err != nil {
			log.Fatalf("Failed bartime2minute : %s, %v", t, err)
		} else {
			m = int32(i)
		}
	}
	return m
}

// 输入："%H:%M:%S"，
// 输出：M值每个周期的整分钟值
func bartime2Minute(t string, period BfBarPeriod) int32 {
	if x, ok := periodMinutesList[period]; ok {
		return _bartime2Minute(t) / x * x
	} else {
		panic("Bartime2Minute: period not supported.")
	}
}

// 输入："%H:%M:%S"
// 输出：H值
func bartime2Hour(t string) int32 {
	var h int32
	if strings.Count(t, ":") != 2 {
		log.Fatalf("Failed bartime2minute : %s", t)
	}
	start := strings.Index(t, ":")
	if start > 0 {
		i, err := strconv.Atoi(t[:start])
		if err != nil {
			log.Fatalf("Failed bartime2minute : %s, %v", t, err)
		} else {
			h = int32(i)
		}
	}
	return h
}

// 输入：两个时间值（不包含日期）与周期
// 输出：是否属于同一个周期
func isSamePeriodTime(previous string, current string, period BfBarPeriod) bool {
	// 只支持分钟与小时，日要用日期而不是时间
	//log.Printf("IsNewPeriod:%v, %v", previous, current)
	if period == BfBarPeriod_PERIOD_H01 {
		return bartime2Hour(previous) == bartime2Hour(current)
	} else {
		// 多分钟的
		t1 := bartime2Minute(previous, period)
		t2 := bartime2Minute(current, period)
		return t1 == t2
	}
	panic("unknow period")
}

// 用Tick数据构造一个新Bar并返回
func NewBarFromTick(t *BfTickData, period BfBarPeriod) *BfBarData {
	b := &BfBarData{Period: period}
	b.Symbol = t.Symbol
	b.Exchange = t.Exchange

	b.ActionDate = t.ActionDate
	b.BarTime = ticktime2Bartime(t.TickTime, period) //"%H:%M:%S.%f"==>"%H:%M:%S"
	b.Volume = t.Volume
	b.OpenInterest = t.OpenInterest
	b.LastVolume = t.LastVolume

	b.OpenPrice = t.LastPrice
	b.HighPrice = t.LastPrice
	b.LowPrice = t.LastPrice
	b.ClosePrice = t.LastPrice

	return b
}

// 用Tick数据更新一个已有Bar
func updateBarFromTick(b *BfBarData, t *BfTickData) {
	b.BarTime = ticktime2Bartime(t.TickTime, b.Period) //"%H:%M:%S.%f"==>"%H:%M:%S"

	b.HighPrice = math.Max(b.HighPrice, t.LastPrice)
	b.LowPrice = math.Min(b.LowPrice, t.LastPrice)
	b.ClosePrice = t.LastPrice

	b.Volume = t.Volume
	b.OpenInterest = t.OpenInterest
	b.LastVolume += t.LastVolume
}

// 将tick更新到传入的某周期的bar
// 返回值
// bool：是否新周期开始
// *BfBarData：如果新周期开始，返回的是新生成的新周期的bar；否则是空
func UpdateTick2Bar(tick *BfTickData, bar *BfBarData) (*BfBarData, bool) {
	var ret *BfBarData = nil
	isSamePeriod := true

	if tick == nil || bar == nil || bar.Symbol != tick.Symbol || bar.Exchange != tick.Exchange {
		panic("illegal param")
	}

	period := bar.Period
	// 判断是否新的周期
	if period == BfBarPeriod_PERIOD_D01 {
		isSamePeriod = bar.ActionDate == tick.ActionDate
	} else if period == BfBarPeriod_PERIOD_W01 {
		panic("TODO: WEEK BAR not support")
	} else {
		isSamePeriod = isSamePeriodTime(bar.BarTime, ticktime2Bartime(tick.TickTime, period), period)
	}

	if isSamePeriod {
		// 还在同一个周期中，更新即可
		updateBarFromTick(bar, tick)
	} else {
		// 新的周期开始，需要生成新周期的bar返回
		// 用tick初始化一个新的currentBar
		ret = NewBarFromTick(tick, period)
	}

	return ret, !isSamePeriod
}
