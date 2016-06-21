package main

import (
	"log"
	"time"
)
import . "github.com/sunwangme/bfgo/api/bfgateway"
import . "github.com/sunwangme/bfgo/api/bfdatafeed"

// 本策略感兴趣的周期，必须是bar.PeriodKeyList的一个子集！
var myPeriodKeyList = []BfBarPeriod{
	BfBarPeriod_PERIOD_M15}

type Trader struct {
	client *BfClient
}

func NewTrader(client *BfClient) *Trader {
	return &Trader{client: client}
}
func (*Trader) PeriodKeyList() []BfBarPeriod {
	return myPeriodKeyList
}
func (*Trader) OnTick(*DataFrames) {
	//×××××××××金十动力策略××××××××××
	//【简介】开仓后持有10周期的策略！
	//【状态】15分钟偏离60均线1%以上
	//【开仓】15分macd红/绿开仓
	//【平仓】绿/红平1/3，10周期平1/3，60均线平1/3
}
func (*Trader) OnTrade(*BfTradeData) {
}

// Order方向分：开仓，平仓
type DirectionType int

const (
	DIRECTION_OPEN  DirectionType = 0
	DIRECTION_CLOSE DirectionType = 1
)

type Task struct {
	order    *BfSendOrderReq
	position *BfPositionData
	t        DirectionType
}

func NewTask10pOpen(order *BfSendOrderReq, position *BfPositionData) *Task {
	return &Task{order: order, position: position, t: DIRECTION_OPEN}
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
