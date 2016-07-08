package main

import (
	"log"
	"time"
)
import . "github.com/sunwangme/bfgo/api/bfgateway"
import . "github.com/sunwangme/bfgo/api/bfdatafeed"
import "github.com/oneywang/bfgo/qite"

// 任一策略对当前周期的趋势方向判断结果有三：上升，下跌，震荡
type TrendStateType int

const (
	TRENDSTATE_BULL   TrendStateType = 0
	TRENDSTATE_BEAR   TrendStateType = 1
	TRENDSTATE_MONKEY TrendStateType = 2
)

// 判断a线是否上穿b线
func CrossUp(a0, a1, b0, b1 float64) bool {
	return a0 < b0 && a1 > b1
}

// 判断a线是否下穿b线
func CrossDown(a0, a1, b0, b1 float64) bool {
	return a0 > b0 && a1 < b1
}

//【状态】15分钟K的当前趋势状态：偏离60均线1%以上算BULL，以下算BEAR
func M15State(d *qite.BarSeries) TrendStateType {
	// TODO:
	return TRENDSTATE_MONKEY
}

// 本策略感兴趣的周期，必须是bar.PeriodKeyList的一个子集！
var myPeriodKeyList = []BfBarPeriod{
	BfBarPeriod_PERIOD_M05} //TODO: M15

type Trader struct {
	client        *BfClient
	opener        *OpenTask
	closer1       *CloseTask
	closer2       *CloseTask
	closer3       *CloseTask
	pendingOrders []string
}

func NewTrader(client *BfClient) *Trader {
	t := &Trader{client: client}
	//TODO:
	t.opener = NewTask10pOpen(t, "15分macd红/绿开仓", 5)
	t.closer1 = NewTask10pClose("红变绿/绿变红平1/3")
	t.closer1 = NewTask10pClose("10周期平1/3")
	t.closer1 = NewTask10pClose("穿60均线平1/3")
	return t
}
func (*Trader) PeriodKeyList() []BfBarPeriod {
	return myPeriodKeyList
}

func (p *Trader) OnTick(dataframes *qite.DataFrames) {
	//×××××××××金十动力策略××××××××××
	//【简介】开仓后持有10周期的策略！
	bars := (*dataframes)[BfBarPeriod_PERIOD_M15]
	if !bars.Enough() {
		log.Print("Not enough bars")
		return
	}
	//【状态】15分钟偏离60均线1%以上
	M15State(bars)
	//【开仓】15分macd红/绿开仓
	p.opener.Run(bars)
	//【平仓】绿/红平1/3，10周期平1/3，60均线平1/3
	p.closer1.Run(bars)
	p.closer2.Run(bars)
	p.closer3.Run(bars)
}

func (p *Trader) Buy(price float64, vol int32) string {
	orderid := buy(p.client, price, vol)
	p.pendingOrders = append(p.pendingOrders, orderid)
	return orderid
}

func (p *Trader) Short(price float64, vol int32) string {
	orderid := short(p.client, price, vol)
	p.pendingOrders = append(p.pendingOrders, orderid)
	return orderid
}

func (p *Trader) OnTrade(resp *BfTradeData) {
	if indexOf(p.pendingOrders, resp.BfOrderId) == -1 {
		// TODO：不是本策略本次运行发起的交易
		return
	}
	// 按最新成交结果：1.更新orderids, 2.更新当前仓位
	p.pendingOrders = without(p.pendingOrders, resp.BfOrderId)
	updatePosition(resp.Direction, resp.Offset, resp.Volume)
}

func buy(client *BfClient, price float64, volume int32) string {
	log.Printf("%v", time.Now())
	log.Printf("To Buy: price=%10.3f vol=%d", price, volume)
	resp, err := client.SendOrder(&BfSendOrderReq{
		Symbol:    client.symbol,
		Exchange:  client.exchange,
		Price:     price,
		Volume:    volume,
		PriceType: BfPriceType_PRICETYPE_LIMITPRICE,
		Direction: BfDirection_DIRECTION_LONG,
		Offset:    BfOffset_OFFSET_OPEN})
	if err != nil {
		log.Fatal("Buy error")
	}

	return resp.BfOrderId
}

func sell(client *BfClient, price float64, volume int32) string {
	log.Printf("%v", time.Now())
	log.Printf("To sell: price=%10.3f vol=%d", price, volume)
	resp, err := client.SendOrder(&BfSendOrderReq{
		Symbol:    client.symbol,
		Exchange:  client.exchange,
		Price:     price,
		Volume:    volume,
		PriceType: BfPriceType_PRICETYPE_LIMITPRICE,
		Direction: BfDirection_DIRECTION_LONG,
		Offset:    BfOffset_OFFSET_CLOSETODAY})
	if err != nil {
		log.Fatal("sell error")
	}

	return resp.BfOrderId
}

func short(client *BfClient, price float64, volume int32) string {
	log.Printf("%v", time.Now())
	log.Printf("short: price=%10.3f vol=%d", price, volume)
	resp, err := client.SendOrder(&BfSendOrderReq{
		Symbol:    client.symbol,
		Exchange:  client.exchange,
		Price:     price,
		Volume:    volume,
		PriceType: BfPriceType_PRICETYPE_LIMITPRICE,
		Direction: BfDirection_DIRECTION_SHORT,
		Offset:    BfOffset_OFFSET_OPEN})
	if err != nil {
		log.Fatal("short error")
	}
	return resp.BfOrderId
}

func cover(client *BfClient, price float64, volume int32) string {
	log.Printf("%v", time.Now())
	log.Printf("To cover: price=%10.3f vol=%d", price, volume)
	resp, err := client.SendOrder(&BfSendOrderReq{
		Symbol:    client.symbol,
		Exchange:  client.exchange,
		Price:     price,
		Volume:    volume,
		PriceType: BfPriceType_PRICETYPE_LIMITPRICE,
		Direction: BfDirection_DIRECTION_SHORT,
		Offset:    BfOffset_OFFSET_CLOSETODAY})
	if err != nil {
		log.Fatal("cover error")
	}
	return resp.BfOrderId
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
