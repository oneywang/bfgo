package qite

import (
	"log"
	//"math"
	//"time"
)
import . "github.com/sunwangme/bfgo/api/bfgateway"
import . "github.com/sunwangme/bfgo/api/bfdatafeed"

import "github.com/go-talib"

// PriceType是OHCL一种
type PriceType int

const (
	PRICETYPE_OPEN  PriceType = 0
	PRICETYPE_HIGH  PriceType = 1
	PRICETYPE_LOW   PriceType = 2
	PRICETYPE_CLOSE PriceType = 3
)

// 参数常量
const (
	GOLDTEN_MIN_K_NUM int = 60 //计算时至少需要多少根K线
)

type DataFrames map[BfBarPeriod]*BarSeries

type BarSeries struct {
	// 不同品种K线
	period                 BfBarPeriod
	timeStart              string
	timeCurrent            string
	bars                   []*BfBarData
	macd, ma15, ma30, ma60 []float64
}

//分配某周期的dataframe
func NewBarSeries(p BfBarPeriod, t string) *BarSeries {
	return &BarSeries{period: p, timeStart: t}
}

func price(b *BfBarData, t PriceType) float64 {
	switch t {
	case PRICETYPE_OPEN:
		return b.OpenPrice
	case PRICETYPE_HIGH:
		return b.HighPrice
	case PRICETYPE_LOW:
		return b.LowPrice
	case PRICETYPE_CLOSE:
		return b.ClosePrice
	}
	panic("unkonw price type")
}

func prices(b []*BfBarData, t PriceType) []float64 {
	count := len(b)
	p := make([]float64, count)
	for i, v := range b {
		p[i] = price(v, t)
	}
	return p
}

//增加一行数据，同时计算macd ma15/30/60；初始补数据用
func (p *BarSeries) AppendBar(b *BfBarData) {
	if b != nil {
		log.Printf("apped bar: %v", b)
		p.bars = append(p.bars, b)
		if len(p.bars) > GOLDTEN_MIN_K_NUM*5 {
			// 当bar的数目超过5倍最少计算值时，丢弃老的一部分
			p.bars = p.bars[GOLDTEN_MIN_K_NUM:]
		}
	}
	//是否积累了足够多的k线
	count := p.Count()
	if count < GOLDTEN_MIN_K_NUM {
		log.Print("more bar needed to calc ma/macd")
		return
	}

	//计算MA15,30,60值，MACD值
	closePrices := prices(p.bars, PRICETYPE_CLOSE)
	p.ma15 = talib.Sma(closePrices, 15)
	p.ma30 = talib.Sma(closePrices, 30)
	p.ma60 = talib.Sma(closePrices, 60)
	_, _, p.macd = talib.Macd(closePrices, 12, 26, 9)
}

//盘中推tick用
//更新或增加一行数据，同时计算macd ma15/30/60；
//返回是否新bar产生；
func (p *BarSeries) AppendTick(t *BfTickData) (*BfBarData, bool) {
	count := p.Count()
	bar, hasNew := UpdateTick2Bar(t, p.bars[count-1])
	p.AppendBar(bar)
	return bar, hasNew
}

//bar个数
func (p *BarSeries) Count() int {
	return len(p.bars)
}

func (p *BarSeries) Enough() bool {
	return p.Count() >= GOLDTEN_MIN_K_NUM
}

//bar
func (p *BarSeries) Bar(index int) (*BfBarData, bool) {
	var ret *BfBarData = nil
	ok := false
	if index >= 0 && index < p.Count() {
		ret = p.bars[index]
		ok = true
	}
	return ret, ok
}

//获取最新macd,priceType是OHCL一种
func (p *BarSeries) Macd() []float64 {
	return p.macd
}

//获取最新ma15
func (p *BarSeries) Ma15(priceType PriceType) float64 {
	ma := talib.Sma(prices(p.bars, priceType), 15)
	return ma[len(ma)-1]
}

//获取最新ma30
func (p *BarSeries) Ma30(priceType PriceType) float64 {
	ma := talib.Sma(prices(p.bars, priceType), 30)
	return ma[len(ma)-1]
}

//获取最新ma60
func (p *BarSeries) Ma60(priceType PriceType) float64 {
	ma := talib.Sma(prices(p.bars, priceType), 60)
	return ma[len(ma)-1]
}

//最高价
func (p *BarSeries) Max(priceType PriceType) float64 {
	m := talib.Max(prices(p.bars, priceType), p.Count())
	return m[len(m)-1]
}

//最低价
func (p *BarSeries) Min(priceType PriceType) float64 {
	m := talib.Min(prices(p.bars, priceType), p.Count())
	return m[len(m)-1]
}

//释放
//df.free()
