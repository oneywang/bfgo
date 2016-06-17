package main

import (
	"log"
	"math"
	"strconv"
	"strings"
)
import . "github.com/sunwangme/bfgo/api/bfgateway"
import . "github.com/sunwangme/bfgo/api/bfdatafeed"

var periodKeyList = []BfBarPeriod{
	BfBarPeriod_PERIOD_M03,
	BfBarPeriod_PERIOD_M05,
	BfBarPeriod_PERIOD_M10,
	BfBarPeriod_PERIOD_M15,
	BfBarPeriod_PERIOD_M30}

var periodMinutesList = map[BfBarPeriod]int32{
	BfBarPeriod_PERIOD_M03: 3,
	BfBarPeriod_PERIOD_M05: 5,
	BfBarPeriod_PERIOD_M10: 10,
	BfBarPeriod_PERIOD_M15: 15,
	BfBarPeriod_PERIOD_M30: 30}

const (
	BFTICKTIMELAYOUT string = "2006010215:04:05.000"
	BFBARTIMELAYOUT  string = "2006010215:04:00"
)

// "%H:%M:%S.%f"==>"%H:%M:%S"
func Ticktime2Bartime(t string) string {
	var b string
	//	if tt, err := time.Parse(t, BFTICKTIMELAYOUT); err != nil {
	//		b = tt.Format(BFBARTIMELAYOUT)
	//		}

	if dot := strings.LastIndex(t, "."); dot >= 0 {
		b = t[:dot]
	} else {
		log.Fatalf("Failed ticktime2bartime : %s", t)
	}
	return b
}

// "%H:%M:%S"
func Bartime2Minute(t string) int32 {
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

// "%H:%M:%S"
func Bartime2Hour(t string) int32 {
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

type BarSlice map[BfBarPeriod]*BfBarData

type Bars struct {
	// 不同品种当前的1分钟K线
	data map[string]*BarSlice

	// 要把contract保存到datafeed里面才会看到数据
	// 判断是否初始化了这个保存动作的标志
	contractInited bool
}

func Barxxtime2Minute(t string, period BfBarPeriod) int32 {
	if x, ok := periodMinutesList[period]; ok {
		return Bartime2Minute(t) / x * x
	} else {
		panic("Bartime2Minute: period not supported.")
	}
}

func IsSamePeriodTime(previous string, current string, period BfBarPeriod) bool {
	// TODO：1分钟，小时，日，周特殊处理
	//log.Printf("IsNewPeriod:%v, %v", previous, current)
	if period == BfBarPeriod_PERIOD_M01 {
		t1 := Bartime2Minute(previous)
		t2 := Bartime2Minute(current)
		//log.Printf("Minute:%v, %v", t1, t2)
		return t1 == t2
	} else if period == BfBarPeriod_PERIOD_H01 {
		return Bartime2Hour(previous) == Bartime2Hour(current)
	} else {
		// 多分钟的
		t1 := Barxxtime2Minute(previous, period)
		t2 := Barxxtime2Minute(current, period)
		return t1 == t2
	}
	panic("unknow period")
}

// 用Tick数据赋值Bar
func Tick2M01(t *BfTickData) *BfBarData {
	b := BfBarData{}
	b.Symbol = t.Symbol
	b.Exchange = t.Exchange
	b.Period = BfBarPeriod_PERIOD_M01

	b.ActionDate = t.ActionDate
	b.BarTime = Ticktime2Bartime(t.TickTime) //"%H:%M:%S.%f"==>"%H:%M:%S"
	b.Volume = t.Volume
	b.OpenInterest = t.OpenInterest
	b.LastVolume = t.LastVolume

	b.OpenPrice = t.LastPrice
	b.HighPrice = t.LastPrice
	b.LowPrice = t.LastPrice
	b.ClosePrice = t.LastPrice

	return &b
}

func (p *Bars) UpdatM01(id string, tick *BfTickData) (*BfBarData, bool) {
	var ret *BfBarData = nil
	needInsert := false
	if bm01, ok := p.data[id]; !ok {
		// 这个品种第一次赋值&1分钟的第一次赋值，先生成barSlice与bar
		var bs BarSlice = make(map[BfBarPeriod]*BfBarData)
		bs[BfBarPeriod_PERIOD_M01] = Tick2M01(tick)
		p.data[id] = &bs

	} else {
		storedBar, ok := (*bm01)[BfBarPeriod_PERIOD_M01]
		//log.Printf("%v", storedBar)
		if !ok {
			panic("imposible: no m01 bar")
		}
		// 判断是否新的周期
		tt := Ticktime2Bartime(tick.TickTime)
		if IsSamePeriodTime(storedBar.BarTime, tt, BfBarPeriod_PERIOD_M01) {
			// 还在同一分钟中，更新即可
			//log.Print("is same: update only")
			storedBar.BarTime = tt
			storedBar.HighPrice = math.Max(storedBar.HighPrice, tick.LastPrice)
			storedBar.LowPrice = math.Min(storedBar.LowPrice, tick.LastPrice)
			storedBar.ClosePrice = tick.LastPrice
			storedBar.Volume = tick.Volume
			storedBar.OpenInterest = tick.OpenInterest
			storedBar.LastVolume += tick.LastVolume
		} else {
			// 新的一分钟开始，需要先前的一分钟bar插入了
			log.Print("not same 1min: insert and update")
			needInsert = true
			ret = storedBar
			// 用tick初始化一个新的currentBar
			(*bm01)[BfBarPeriod_PERIOD_M01] = Tick2M01(tick)
		}
	}

	return ret, needInsert
}

// 用快周期的bar得到慢周期的bar
// 返回值
// bool：是否新周期开始
// []*BfBarData：如果新周期开始，返回上一周期的bar以便保持到db去
func (p *Bars) UpdateMxHDW(id string, bar *BfBarData, period BfBarPeriod) ([]*BfBarData, bool) {
	rets := []*BfBarData{}
	needInsert := false

	d, ok := p.data[id]
	if !ok {
		panic("imposible")
	}

	tempBar := BfBarData{}
	if storedBar, ok := (*d)[period]; !ok {
		// 这个周期的bar第一次赋值
		tempBar = *bar
		tempBar.Period = period
		(*d)[period] = &tempBar
	} else {
		// 判断是否新的周期
		isSamePeriod := true
		if period == BfBarPeriod_PERIOD_D01 {
			isSamePeriod = storedBar.ActionDate == bar.ActionDate
		} else if period == BfBarPeriod_PERIOD_W01 {
			isSamePeriod = true //TODO
		} else {
			isSamePeriod = IsSamePeriodTime(storedBar.BarTime, bar.BarTime, period)
		}
		if isSamePeriod {
			// 还在同一个周期中，更新即可
			storedBar.BarTime = bar.BarTime
			storedBar.Volume = bar.Volume
			storedBar.OpenInterest = bar.OpenInterest
			storedBar.LastVolume += bar.LastVolume
			storedBar.HighPrice = math.Max(bar.HighPrice, storedBar.HighPrice)
			storedBar.LowPrice = math.Min(bar.LowPrice, storedBar.LowPrice)
			storedBar.ClosePrice = bar.ClosePrice
		} else {
			// 新的周期开始，需要插入了
			log.Print("not same xxmin: insert and update")
			needInsert = true
			tempBar = *storedBar
			rets = append(rets, &tempBar)

			// 用1分钟的bar初始化一个新的Bar
			*storedBar = *bar //这是深拷贝
			storedBar.Period = period

			// 构造更高级别的bar
			if period == BfBarPeriod_PERIOD_M30 {
				// 30分钟构造小时bar
				hourRets, _ := p.UpdateMxHDW(id, storedBar, BfBarPeriod_PERIOD_H01)
				for i := range hourRets {
					rets = append(rets, hourRets[i])
				}
				// 30分钟同时构造日bar
				dayRets, _ := p.UpdateMxHDW(id, storedBar, BfBarPeriod_PERIOD_D01)
				for i := range dayRets {
					rets = append(rets, dayRets[i])
				}
			} else if period == BfBarPeriod_PERIOD_D01 {
				// 日构造周bar
				weekRets, _ := p.UpdateMxHDW(id, storedBar, BfBarPeriod_PERIOD_W01)
				for i := range weekRets {
					rets = append(rets, weekRets[i])
				}
			}
		}
	}

	return rets, needInsert
}
