package main

import (
	"log"
	"math"
	"time"
)
import . "github.com/sunwangme/bfgo/api/bfgateway"
import . "github.com/sunwangme/bfgo/api/bfdatafeed"
import "github.com/go-talib"

// 支持这些周期的bar计算
var periodKeyList = []BfBarPeriod{
	BfBarPeriod_PERIOD_M01,
	BfBarPeriod_PERIOD_M03,
	BfBarPeriod_PERIOD_M15,
	BfBarPeriod_PERIOD_H01}
var periodMinutesList = map[BfBarPeriod]int32{
	BfBarPeriod_PERIOD_M01: 1,
	BfBarPeriod_PERIOD_M03: 3,
	BfBarPeriod_PERIOD_M15: 15}

type DataFrame struct {
	// 不同品种K线
	period      BfBarPeriod
	timeStart   string
	timeCurrent string
	bars        []*BfBarData
}

//分配某周期的dataframe
func newDataframe(p BfBarPeriod, t string) *DataFrame {
	r := &DataFrame{period: p, timeStart: t}
	return r
}

//增加一行数据，同时计算macd ma15/30/60；初始补数据用
func (p *DataFrame) appendBar(b *BfBarData) {
	p.bars = append(p.bars, b)
	//TODO
}

//增加一行数据，同时计算macd ma15/30/60；
//返回是否新bar产生；盘中推tick用
func (p *DataFrame) appendTick(t *BfTickData) (*BfBarData, bool) {
	var ret *BfBarData = nil
	newBar := false
	//TODO
	return ret, newBar
}

//bar个数
func (p *DataFrame) count() int {
	return len(p.bars)
}

//bar
func (p *DataFrame) bar(index int) (*BfBarData, bool) {
	var ret *BfBarData = nil
	ok := false
	if index >= 0 && index < len(p.bars) {
		ret = p.bars[index]
		ok = true
	}
	return ret, true
}

//获取最新macd,priceType是OHCL一种
func (p *DataFrame) macd(priceType BfPriceType) float64 {
	ret := 0.9
	return ret
}

//获取最新ma15
func (p *DataFrame) ma15(priceType BfPriceType) float64 {
	ret := 0.15
	return ret
}

//获取最新ma30
func (p *DataFrame) ma30(priceType BfPriceType) float64 {
	ret := 0.30
	return ret
}

//获取最新ma60
func (p *DataFrame) ma60(priceType BfPriceType) float64 {
	ret := 0.60
	return ret
}

//最高价
func (p *DataFrame) max(priceType BfPriceType) float64 {
	ret := 0.99
	return ret
}

//最低价
func (p *DataFrame) min(priceType BfPriceType) float64 {
	ret := 0.01
	return ret
}

//释放
//df.free()

//【副总】和哥v5-招全栈工程师 2016-6-18 20:39:47
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
