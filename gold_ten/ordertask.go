package main

import . "github.com/sunwangme/bfgo/api/bfgateway"
import . "github.com/sunwangme/bfgo/api/bfdatafeed"
import "github.com/oneywang/bfgo/qite"

// Order（Task）方向分：开仓，平仓
type DirectionType int

const (
	DIRECTION_OPEN  DirectionType = 0
	DIRECTION_CLOSE DirectionType = 1
)

const POSITION_LIMIT = 5 //最多持有5手：全空或全多或多空

//****************
//*****开仓任务*****
//****************
type OpenTask struct {
	trader           *Trader
	name             string
	direction        DirectionType
	posBuy, posShort int32
	startBar         *BfBarData
}

func NewTask10pOpen(t *Trader, n string, limit int32) *OpenTask {
	return &OpenTask{trader: t, name: n, direction: DIRECTION_OPEN}
}
func (this *OpenTask) Run(d *qite.BarSeries) {
	//【开仓】15分macd红/绿开仓

	if this.posBuy != 0 && this.posShort != 0 {
		// 开了仓等清仓后才会再开仓
		// TODO：不支持加仓或一次没全成交后的补仓
		return
	}
	macd := d.Macd()
	len := len(macd)
	bar, _ := d.Bar(d.Count() - 1)
	price := bar.ClosePrice
	if CrossUp(macd[len-2], macd[len-1], 0, 0) {
		// 绿变红，开多仓
		if this.posBuy == 0 {
			trader.Buy(price, POSITION_LIMIT)
			this.startBar = bar
		}
	} else if CrossDown(macd[len-2], macd[len-1], 0, 0) {
		// 红变绿，开空仓
		if this.posShort == 0 {
			trader.Short(price, POSITION_LIMIT)
			this.startBar = bar
		}
	}
}

//****************
//*****平仓任务*****
//****************
type CloseTask struct {
	name     string
	order    *BfSendOrderReq
	position *BfPositionData
	t        DirectionType
}

func NewTask10pClose(n string) *CloseTask {
	return &CloseTask{name: n, t: DIRECTION_CLOSE}
}
func (*CloseTask) Run(d *qite.BarSeries) {
}
