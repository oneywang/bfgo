package main

import (
	"log"
	//"math"
	//"time"
)
import . "github.com/sunwangme/bfgo/api/bfgateway"
import . "github.com/sunwangme/bfgo/api/bfdatafeed"
import "github.com/sunwangme/bfgo/oneywang/bar"

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

type DataFrames map[BfBarPeriod]*DataFrame

type DataFrame struct {
	// 不同品种K线
	period      BfBarPeriod
	timeStart   string
	timeCurrent string
	bars        []*BfBarData
	ma15        []float64
	ma30        []float64
	ma60        []float64
}

//分配某周期的dataframe
func NewDataframe(p BfBarPeriod, t string) *DataFrame {
	r := &DataFrame{period: p, timeStart: t}
	return r
}

//增加一行数据，同时计算macd ma15/30/60；初始补数据用
func (p *DataFrame) AppendBar(b *BfBarData) {
	if b != nil {
		p.bars = append(p.bars, b)
	}
	//是否积累了足够多的k线
	count := p.Count()
	if count < GOLDTEN_MIN_K_NUM {
		log.Print("more bar needed to calc ma")
		return
	}
	//计算MA值
	closePrices := make([]float64, count)
	for i := range p.bars {
		closePrices[i] = p.bars[i].ClosePrice
	}
	p.ma15 = talib.Sma(closePrices, 15)
	p.ma30 = talib.Sma(closePrices, 30)
	p.ma60 = talib.Sma(closePrices, 60)
}

//盘中推tick用
//更新或增加一行数据，同时计算macd ma15/30/60；
//返回是否新bar产生；
func (p *DataFrame) AppendTick(t *BfTickData) (*BfBarData, bool) {
	count := p.Count()
	bar, hasNew := bar.UpdateTick2Bar(t, p.bars[count-1])
	p.AppendBar(bar)
	return bar, hasNew
}

//bar个数
func (p *DataFrame) Count() int {
	return len(p.bars)
}

//bar
func (p *DataFrame) Bar(index int) (*BfBarData, bool) {
	var ret *BfBarData = nil
	ok := false
	if index >= 0 && index < len(p.bars) {
		ret = p.bars[index]
		ok = true
	}
	return ret, ok
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
	for i := range b {
		p[i] = price(b[i], t)
	}
	return p
}

//获取最新macd,priceType是OHCL一种
func (p *DataFrame) Macd(priceType PriceType) float64 {
	count := p.Count()
	if count < GOLDTEN_MIN_K_NUM {
		log.Print("more bar needed to calc ma")
		return 0
	}

	macd, _, _ := talib.Macd(prices(p.bars, priceType), 12, 26, 9)
	return macd[count-1]
}

//获取最新ma15
func (p *DataFrame) Ma15(priceType PriceType) float64 {
	ma := talib.Sma(prices(p.bars, priceType), 15)
	return ma[len(ma)-1]
}

//获取最新ma30
func (p *DataFrame) Ma30(priceType PriceType) float64 {
	ma := talib.Sma(prices(p.bars, priceType), 30)
	return ma[len(ma)-1]
}

//获取最新ma60
func (p *DataFrame) Ma60(priceType PriceType) float64 {
	ma := talib.Sma(prices(p.bars, priceType), 60)
	return ma[len(ma)-1]
}

//最高价
func (p *DataFrame) Max(priceType PriceType) float64 {
	m := talib.Max(prices(p.bars, priceType), p.Count())
	return m[len(m)-1]
}

//最低价
func (p *DataFrame) Min(priceType PriceType) float64 {
	m := talib.Min(prices(p.bars, priceType), p.Count())
	return m[len(m)-1]
}

//释放
//df.free()

//【和哥v5】 2016-6-18 20:39:47
//. 策略例子
//onstart
//   初始化各周期dataframe如dfs

//   补最近60根k到dfs
//ontick
//   推tick到dfs

//   捕捉行情任务运行一次，基于order的，order成交后，要明确定义止盈止损条件，形成止盈止损任务
//   TradeTaskRunOnce(dfs)

//   止盈止损任务运行一次，基于order的，而不是基于pos
//   StopTaskRunOnce(dfs)

//我做策略，都是跟踪order的，order只有被平了，才算完成，也就是一次交易。
//非常不喜欢基于pos的。
//每一个order，定义止盈止损算法和参数，然后就形成一个任务了
//而不是基于总体pos来止盈止损，那个太恶心了
//好了，秘籍今天就说这么多
//**不懂吧？举个栗子
//比如我3分钟背离红多10 加仓10 加仓10，就是3个单子，然后是15分钟的macd红，加仓30，这四个单子的止盈止损不一样
//不是基于总体仓位平今价 盈利水平来的，而是基于当初那个单子的目标下的
//比如3分钟的止盈止损，是10个周期；15分钟的是10个周期；必须单独弄！
//也方便写程序--模块化，逻辑清晰
//就是一个单子的生命周期跟踪，从开 成交 到 平仓，而不是基于总pos
//下一个单子，就必须给这个单子设置止盈止损算法和参数，然后就是跟踪这个单子。
//总体就是两个基于dataframe的模块，一个是产生order前的，一个是产生order后
